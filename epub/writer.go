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

	TmplPath = "tmpl"
)

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

type Writer struct {
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

func NewWriter(w io.Writer, identifier string) (*Writer, error) {
	nameSpace := uuid.NewSHA1(uuid.NameSpaceURL, []byte(baseNameSpaceURL))

	zipFile := zip.NewWriter(w)
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

	tmplDir, err := filepath.Abs(TmplPath)
	if err != nil {
		return nil, err
	}

	epub := &Writer{
		UUID:         uuid.NewSHA1(nameSpace, []byte(identifier)),
		LastModified: time.Now().UTC().Format(time.RFC3339),
		Language:     "en-GB",

		open:  true,
		zip:   zipFile,
		Files: make(map[string]*File),

		tmplDir: tmplDir,
	}

	nav := epub.RegisterFile(navName, "application/xhtml+xml", false)
	epub.NavPath = nav.Path
	css := epub.RegisterFile(cssName, "text/css", false)
	epub.CSSPath = css.Path

	return epub, nil
}

func (epub *Writer) Flush() error {
	if !epub.open {
		return nil
	}

	err := epub.closeSections(0)
	if err != nil {
		return err
	}

	k := len(epub.Nav) - 1
	if k >= 0 {
		epub.Nav[k].down = epub.Nav[k].Level
	}

	epub.ContentName = contentDir + epub.uniqueName(contentName, contentExt)

	files := []struct {
		path      string
		templates []string
	}{
		{contentDir + epub.Files[epub.CSSPath].Path,
			[]string{"book.css"}},
		{contentDir + epub.Files[epub.NavPath].Path,
			[]string{"nav.xhtml", "config/epub"}},
		{epub.ContentName,
			[]string{"content.opf"}},
		{containerName,
			[]string{"container.xml"}},
	}
	for _, file := range files {
		err := epub.addFileFromTemplate(file.path, file.templates, nil)
		if err != nil {
			return err
		}
	}

	epub.zip.Close()
	epub.open = false
	return nil
}

func (epub *Writer) RegisterFile(name, mimeType string, inSpine bool) *File {
	file := &File{
		ID:        "f" + strconv.Itoa(epub.nextID),
		MediaType: mimeType,
	}
	epub.nextID++

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
	file.Path = epub.uniqueName(dir+name, ext)

	epub.Files[file.Path] = file
	if inSpine {
		epub.Spine = append(epub.Spine, file)
	}
	return file
}

func (epub *Writer) CreateFile(file *File) (io.Writer, error) {
	err := epub.closeSections(0)
	if err != nil {
		return nil, err
	}
	return epub.zip.Create(contentDir + file.Path)
}

func (epub *Writer) AddCoverImage(r io.Reader) error {
	if !epub.open {
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

	coverImage := epub.RegisterFile(coverName, mimeType, false)
	fd, err := epub.CreateFile(coverImage)
	if err != nil {
		return err
	}
	_, err = io.Copy(fd, rBuf)
	if err != nil {
		return err
	}
	epub.CoverImageID = coverImage.ID

	cover := epub.RegisterFile(coverName, "application/xhtml+xml", true)
	err = epub.addFileFromTemplate(contentDir+cover.Path,
		[]string{"cover.xhtml", "config/epub"},
		map[string]string{
			"CoverImage": coverImage.Path,
		})
	if err != nil {
		return err
	}
	epub.CoverID = cover.ID

	return nil
}

func (epub *Writer) WriteTitle() error {
	if !epub.open {
		return ErrBookClosed
	}
	if epub.Title == "" {
		return ErrNoTitle
	}

	file := epub.RegisterFile(titleName, "application/xhtml+xml", true)
	err := epub.addFileFromTemplate(contentDir+file.Path,
		[]string{"title.xhtml", "config/epub"}, nil)
	if err != nil {
		return err
	}

	return nil
}

func (epub *Writer) closeSections(level int) error {
	if epub.SectionLevel <= level {
		return nil
	}

	for epub.SectionLevel > level {
		err := epub.writeTemplates(epub.current,
			[]string{"section-tail.xhtml"},
			nil)
		if err != nil {
			return err
		}
		epub.SectionLevel--
	}

	if epub.SectionLevel <= 0 {
		err := epub.writeTemplates(epub.current,
			[]string{"chapter-tail.xhtml"},
			nil)
		epub.current = nil
		if err != nil {
			return err
		}
	}
	return nil
}

func (epub *Writer) AddSection(level int, title string, secID string) error {
	if level <= 0 || level > epub.SectionLevel+1 {
		return ErrWrongSectionLevel
	}
	err := epub.closeSections(level - 1)
	if err != nil {
		return err
	}
	epub.SectionLevel = level
	epub.SectionNumber.Inc(level)

	if epub.current == nil {
		name := fmt.Sprintf("ch%s", epub.SectionNumber)
		file := epub.RegisterFile(name, "application/xhtml+xml", true)

		w, err := epub.zip.Create(contentDir + file.Path)
		if err != nil {
			return err
		}
		epub.current = w
		epub.currentPath = file.Path
		err = epub.writeTemplates(epub.current,
			[]string{"chapter-head.xhtml", "config/epub"},
			map[string]interface{}{
				"Level": level,
				"Title": title,
				"EPUB":  epub,
			})
		if err != nil {
			return err
		}
	}

	if secID == "" {
		secID = "epub-" + epub.SectionNumber.String()
	}

	k := len(epub.Nav) - 1
	var up, down int
	if k >= 0 {
		if level < epub.Nav[k].Level {
			down = epub.Nav[k].Level - level
		} else {
			up = level - epub.Nav[k].Level
		}
		epub.Nav[k].down = down
	} else {
		up = level
	}

	epub.Nav = append(epub.Nav, TOCEntry{
		Level: level,
		Title: title,
		Path:  epub.currentPath,
		ID:    secID,
		up:    up,
	})

	return epub.writeTemplates(epub.current,
		[]string{"section-head.xhtml"},
		map[string]interface{}{
			"Level": level,
			"SecNo": epub.SectionNumber,
			"Title": title,
			"EPUB":  epub,
			"ID":    secID,
		})
}

func (epub *Writer) WriteString(s string) error {
	if epub.current == nil {
		return ErrNoSection
	}
	_, err := epub.current.Write([]byte(s))
	return err
}

func (epub *Writer) uniqueName(name, ext string) string {
	tryName := name + ext
	unique := 2
	for {
		_, clash := epub.Files[tryName]
		if !clash {
			break
		}
		tryName = name + strconv.Itoa(unique) + ext
		unique++
	}
	return tryName
}
