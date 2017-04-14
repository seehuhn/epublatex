// driver.go - generalize EPUB and XHTML output
//
// Copyright (C) 2016  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package epub

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

const (
	epubContainerName  = "META-INF/container.xml"
	epubContentDir     = "OEBPS/"
	epubContentExt     = ".opf"
	epubContentName    = "content"
	epubMimeType       = "application/epub+zip"
	epubTemplateConfig = "config/epub"

	xhtmlTemplateConfig = "config/xhtml"
)

// `driver` generalizes epub and xhtml output.
type driver interface {
	Close(w *Book) error
	Create(path string) (io.WriteCloser, error)
	MakePath(path string) string
	Config() string
}

type epubDriver struct {
	ZipFile *zip.Writer
}

func (drv *epubDriver) Close(w *Book) error {
	contentName := epubContentDir + w.uniqueName(epubContentName, epubContentExt)
	err := w.addFileFromTemplate(contentName, []string{"content.opf"}, nil)
	if err != nil {
		return err
	}

	err = w.addFileFromTemplate(epubContainerName, []string{"container.xml"},
		map[string]string{"ContentName": contentName})
	if err != nil {
		return err
	}

	return drv.ZipFile.Close()
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func (drv *epubDriver) Create(path string) (io.WriteCloser, error) {
	out, err := drv.ZipFile.Create(path)
	if err != nil {
		return nil, err
	}
	return nopWriteCloser{out}, nil
}

func (drv *epubDriver) MakePath(path string) string {
	return epubContentDir + path
}

func (drv *epubDriver) Config() string {
	return epubTemplateConfig
}

type xhtmlDriver struct {
	BaseDir string
}

func (drv *xhtmlDriver) Close(w *Book) error {
	return nil
}

func (drv *xhtmlDriver) Create(path string) (io.WriteCloser, error) {
	outPath := filepath.Join(drv.BaseDir, filepath.FromSlash(path))
	outDir := filepath.Dir(outPath)
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(outPath)
}

func (drv *xhtmlDriver) MakePath(path string) string {
	return path
}

func (drv *xhtmlDriver) Config() string {
	return xhtmlTemplateConfig
}
