// writer.go - write generated HTML to the output document
//
// Copyright (C) 2016, 2017  Jochen Voss <voss@seehuhn.de>
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

package latex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seehuhn/epublatex/epub"
)

const outputLineWidth = 79

const cssPrefix = "latex-"

type writer struct {
	out     *epub.Book
	baseDir string

	word       []byte
	line       []string
	lineLength int

	state *State
	stack []*State

	nextParTag string
}

func newWriter(out *epub.Book, baseDir string) *writer {
	return &writer{
		out:     out,
		baseDir: baseDir,
		state:   &State{},
	}
}

func (w *writer) Flush() error {
	return w.EndParagraph()
}

func (w *writer) AddCoverImage(fname string) error {
	fd, err := w.openFile(fname)
	if err != nil {
		return err
	}
	defer fd.Close()
	return w.out.AddCoverImage(fd)
}

func (w *writer) WriteTitle(title, author string) error {
	e1 := w.EndParagraph()
	e2 := w.out.AddTitle(title, []string{author})
	return firstOf(e1, e2)
}

func (w *writer) AddSection(level int, title string, id string) error {
	e1 := w.EndParagraph()
	e2 := w.out.AddSection(level, title, id)
	return firstOf(e1, e2)
}

func (w *writer) WriteVertical(body string) error {
	e1 := w.suspendParagraph("cont")
	e2 := w.out.WriteString(body)
	return firstOf(e1, e2)
}

func (w *writer) StartBlock(name string, classes []string, id string) error {
	w.EndParagraph()

	if id != "" {
		id = ` id="` + id + `"`
	}
	cssClasses := []string{
		cssPrefix + "block",
		name,
	}
	for _, cls := range classes {
		cssClasses = append(cssClasses, cls)
	}
	line := fmt.Sprintf("<div%s class=\"%s\">\n",
		id, strings.Join(cssClasses, " "))
	return w.out.WriteString(line)
}

func (w *writer) EndBlock() error {
	w.EndParagraph()
	return w.out.WriteString("</div>\n")
}

func (w *writer) suspendParagraph(contTag string) error {
	w.nextParTag = ""
	e1 := w.endWord(true)
	if len(w.line) == 0 {
		return e1
	}
	w.nextParTag = contTag
	e2 := w.writeLine()
	return firstOf(e1, e2)
}

func (w *writer) EndParagraph() error {
	return w.suspendParagraph("")
}

func (w *writer) writeLine() error {
	if len(w.line) == 0 {
		return nil
	}

	lineStr := strings.Join(w.line, " ") + "\n"
	w.line = nil
	return w.out.WriteString(lineStr)
}

func (w *writer) endWord(endPar bool) error {
	word := string(w.word)
	w.word = nil

	if strings.Contains(word, noBreakSpace) {
		word = "<span class=\"" + cssPrefix + "nw\">" + word + "</span>"
	}
	if endPar {
		endTag := "</p>"
		if w.state.isBold {
			endTag = "</b>" + endTag
		}
		if w.state.isItalic {
			endTag = "</i>" + endTag
		}
		if word != "" {
			word = word + endTag
		} else if len(w.line) > 0 {
			k := len(w.line) - 1
			w.line[k] = w.line[k] + endTag
			w.lineLength += 4
		}
	}
	l := len(word)
	if l == 0 {
		return nil
	}

	if len(w.line) == 0 {
		tag := "<p>"
		if w.nextParTag != "" {
			tag = `<p class="` + w.nextParTag + `">`
		}
		if w.state.isItalic {
			tag = tag + "<i>"
		}
		if w.state.isBold {
			tag = tag + "<b>"
		}
		w.line = []string{tag + word}
		w.lineLength = len(tag) + l
	} else if w.lineLength+1+l <= outputLineWidth {
		w.line = append(w.line, word)
		w.lineLength += 1 + l
	} else {
		err := w.writeLine()
		if err != nil {
			return err
		}
		w.line = []string{word}
		w.lineLength = l
	}
	return nil
}

func (w *writer) EndWord() error {
	return w.endWord(false)
}

func (w *writer) WriteString(s string) {
	w.word = append(w.word, []byte(s)...)
}

func (w *writer) openFile(fname string) (*os.File, error) {
	fullName := filepath.Join(w.baseDir, fname)
	return os.Open(fullName)
}
