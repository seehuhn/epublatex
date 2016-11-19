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

type epubDriver struct {
	ZipFile *zip.Writer
}

func (drv *epubDriver) Close(w *book) error {
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

func (drv *xhtmlDriver) Close(w *book) error {
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

type driver interface {
	Close(w *book) error
	Create(path string) (io.WriteCloser, error)
	MakePath(path string) string
	Config() string
}
