// writer.go -
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
	"bufio"
	"compress/flate"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	baseNameSpaceURL = "http://ebook.seehuhn.de/"

	mimeType      = "application/epub+zip"
	containerName = "META-INF/container.xml"
	contentDir    = "OEBPS/"
	contentName   = "content"
	contentExt    = ".opf"
	cssName       = "book"
	navName       = "nav"
	coverName     = "cover"
	titleName     = "title"
)

var DefaultTemplateDir = flag.String("epub-templates", "tmpl",
	"directory where the EPUB template files can be found")

var (
	ErrBookClosed        = errors.New("attempt to write in a closed book")
	ErrNoSection         = errors.New("attempt to write outside any section")
	ErrNoTitle           = errors.New("document title not set")
	ErrWrongSectionLevel = errors.New("wrong section level")
	ErrWrongFileType     = errors.New("wrong file type")
)

type File struct {
	ID        string
	MediaType string
	Path      string
}

type Writer interface {
	AddCoverImage(r io.Reader) error
	AddTitle(title string, authors []string) error
	AddSection(level int, title string, secID string) error

	RegisterFile(baseName, mimeType string, inSpine bool) *File
	CreateFile(file *File) (io.Writer, error)
	WriteString(s string) error

	Flush() error
}

type epub struct {
	UUID         uuid.UUID
	LastModified string
	Language     string

	Title   string
	Authors []string

	Spine        []*File
	Files        map[string]*File
	Nav          []TOCEntry
	NavPath      string
	CSSPath      string
	CoverImageID string
	CoverID      string
	ContentName  string

	SectionNumber SecNo
	SectionLevel  int

	open        bool
	zip         *zip.Writer
	nextID      int
	current     io.Writer
	currentPath string
	tmplDir     string
}

type Settings struct {
	TemplateDir string
}

func NewWriter(out io.Writer, identifier string, settings *Settings) (
	Writer, error) {
	nameSpace := uuid.NewSHA1(uuid.NameSpaceURL, []byte(baseNameSpaceURL))

	zipFile := zip.NewWriter(out)
	zipFile.RegisterCompressor(zip.Deflate,
		func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, flate.BestCompression)
		})

	// Write the "mimetype" file.  This must be the first file, and
	// must be uncompressed.
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // no compression
	}
	part, err := zipFile.CreateHeader(header)
	if err != nil {
		return nil, err
	}
	_, err = part.Write([]byte(mimeType))
	if err != nil {
		return nil, err
	}

	tmplDir := *DefaultTemplateDir
	if settings != nil && settings.TemplateDir != "" {
		tmplDir = settings.TemplateDir
	}
	tmplDir, err = filepath.Abs(tmplDir)
	if err != nil {
		return nil, err
	}

	w := &epub{
		UUID:         uuid.NewSHA1(nameSpace, []byte(identifier)),
		LastModified: time.Now().UTC().Format(time.RFC3339),
		Language:     "en-GB",

		open:  true,
		zip:   zipFile,
		Files: make(map[string]*File),

		tmplDir: tmplDir,
	}

	nav := w.RegisterFile(navName, "application/xhtml+xml", false)
	w.NavPath = nav.Path
	css := w.RegisterFile(cssName, "text/css", false)
	w.CSSPath = css.Path

	return w, nil
}

func (w *epub) Flush() error {
	if !w.open {
		return nil
	}

	err := w.closeSections(0)
	if err != nil {
		return err
	}

	k := len(w.Nav) - 1
	if k >= 0 {
		w.Nav[k].down = w.Nav[k].Level
	}

	w.ContentName = contentDir + w.uniqueName(contentName, contentExt)

	files := []struct {
		path      string
		templates []string
	}{
		{contentDir + w.Files[w.CSSPath].Path,
			[]string{"book.css"}},
		{contentDir + w.Files[w.NavPath].Path,
			[]string{"nav.xhtml", "config/epub"}},
		{w.ContentName,
			[]string{"content.opf"}},
		{containerName,
			[]string{"container.xml"}},
	}
	for _, file := range files {
		err := w.addFileFromTemplate(file.path, file.templates, nil)
		if err != nil {
			return err
		}
	}

	w.zip.Close()
	w.open = false
	return nil
}

func (w *epub) RegisterFile(baseName, mimeType string, inSpine bool) *File {
	file := &File{
		ID:        "f" + strconv.Itoa(w.nextID),
		MediaType: mimeType,
	}
	w.nextID++

	dir := ""
	ext := ""
	switch mimeType {
	case "application/xhtml+xml":
		ext = ".xhtml"
	case "text/css":
		ext = ".css"
		dir = "css/"
	case "image/png":
		ext = ".png"
		dir = "img/"
	case "image/jpeg":
		ext = ".jpg"
		dir = "img/"
	default:
		panic("unknown mime type " + mimeType)
	}
	file.Path = w.uniqueName(dir+baseName, ext)

	w.Files[file.Path] = file
	if inSpine {
		w.Spine = append(w.Spine, file)
	}
	return file
}

func (w *epub) createFile(path string) error {
	if w.current != nil {
		err := w.closeFile()
		if err != nil {
			return err
		}
	}
	out, err := w.zip.Create(path)
	if err != nil {
		return err
	}
	w.current = out
	return nil
}

func (w *epub) closeFile() error {
	w.current = nil
	return nil
}

func (w *epub) CreateFile(file *File) (io.Writer, error) {
	if !w.open {
		return nil, ErrBookClosed
	}

	err := w.closeSections(0)
	if err != nil {
		return nil, err
	}
	err = w.createFile(contentDir + file.Path)
	if err != nil {
		return nil, err
	}
	return w.current, nil
}

func (w *epub) AddCoverImage(r io.Reader) error {
	if !w.open {
		return ErrBookClosed
	}

	rBuf := bufio.NewReaderSize(r, 512)
	head, err := rBuf.Peek(512)
	if err != nil {
		return err
	}
	mimeType := http.DetectContentType(head)
	if !strings.HasPrefix(mimeType, "image/") {
		return ErrWrongFileType
	}

	coverImage := w.RegisterFile(coverName, mimeType, false)
	_, err = w.CreateFile(coverImage)
	if err != nil {
		return err
	}
	_, err = io.Copy(w.current, rBuf)
	if err != nil {
		return err
	}
	err = w.closeFile()
	if err != nil {
		return err
	}
	w.CoverImageID = coverImage.ID

	cover := w.RegisterFile(coverName, "application/xhtml+xml", true)
	err = w.addFileFromTemplate(contentDir+cover.Path,
		[]string{"cover.xhtml", "config/epub"},
		map[string]string{
			"CoverImage": coverImage.Path,
		})
	if err != nil {
		return err
	}
	w.CoverID = cover.ID

	return nil
}

func (w *epub) AddTitle(title string, authors []string) error {
	if !w.open {
		return ErrBookClosed
	}

	w.Title = title
	w.Authors = authors
	file := w.RegisterFile(titleName, "application/xhtml+xml", true)
	err := w.addFileFromTemplate(contentDir+file.Path,
		[]string{"title.xhtml", "config/epub"}, nil)
	if err != nil {
		return err
	}

	return nil
}

func (w *epub) closeSections(level int) error {
	if w.SectionLevel <= level {
		return nil
	}

	for w.SectionLevel > level {
		err := w.writeTemplates(w.current,
			[]string{"section-tail.xhtml"},
			nil)
		if err != nil {
			return err
		}
		w.SectionLevel--
	}

	if w.SectionLevel <= 0 {
		err := w.writeTemplates(w.current,
			[]string{"chapter-tail.xhtml"},
			nil)
		if err != nil {
			return err
		}
		err = w.closeFile()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *epub) AddSection(level int, title string, secID string) error {
	if !w.open {
		return ErrBookClosed
	}
	if level <= 0 || level > w.SectionLevel+1 {
		return ErrWrongSectionLevel
	}
	err := w.closeSections(level - 1)
	if err != nil {
		return err
	}
	w.SectionLevel = level
	w.SectionNumber.Inc(level)

	if w.current == nil {
		name := fmt.Sprintf("ch%s", w.SectionNumber)
		file := w.RegisterFile(name, "application/xhtml+xml", true)

		err := w.createFile(contentDir + file.Path)
		if err != nil {
			return err
		}
		w.currentPath = file.Path
		err = w.writeTemplates(w.current,
			[]string{"chapter-head.xhtml", "config/epub"},
			map[string]interface{}{
				"Level": level,
				"Title": title,
			})
		if err != nil {
			return err
		}
	}

	if secID == "" {
		secID = "epub-" + w.SectionNumber.String()
	}

	k := len(w.Nav) - 1
	var up, down int
	if k >= 0 {
		if level < w.Nav[k].Level {
			down = w.Nav[k].Level - level
		} else {
			up = level - w.Nav[k].Level
		}
		w.Nav[k].down = down
	} else {
		up = level
	}

	w.Nav = append(w.Nav, TOCEntry{
		Level: level,
		Title: title,
		Path:  w.currentPath,
		ID:    secID,
		up:    up,
	})

	return w.writeTemplates(w.current,
		[]string{"section-head.xhtml"},
		map[string]interface{}{
			"Level": level,
			"SecNo": w.SectionNumber,
			"Title": title,
			"ID":    secID,
		})
}

func (w *epub) WriteString(s string) error {
	if !w.open {
		return ErrBookClosed
	}
	if w.current == nil {
		return ErrNoSection
	}
	_, err := w.current.Write([]byte(s))
	return err
}

func (w *epub) uniqueName(name, ext string) string {
	tryName := name + ext
	unique := 2
	for {
		_, clash := w.Files[tryName]
		if !clash {
			break
		}
		tryName = name + strconv.Itoa(unique) + ext
		unique++
	}
	return tryName
}
