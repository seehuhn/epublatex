// convert.go -
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

package latex

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/seehuhn/epublatex/epub"
	"github.com/seehuhn/epublatex/latex/math"
)

type converter struct {
	Book *epub.Writer

	SourceDir     string
	WorkDir       string
	TokenFileName string

	Images *math.Images
	Labels []*xRef

	Section  epub.SecNo
	Counters map[string]*counterInfo
	Macros   map[string]macro
	Envs     map[string]*environment
	EnvStack []string

	PkgState map[string]string

	Title, Author string
}

func newConverter(book *epub.Writer) (*converter, error) {
	workDir, err := ioutil.TempDir("", "jvepla")
	if err != nil {
		return nil, err
	}
	conv := &converter{
		Book:    book,
		WorkDir: workDir,

		Macros:   make(map[string]macro),
		Envs:     make(map[string]*environment),
		Counters: make(map[string]*counterInfo),

		PkgState: make(map[string]string),
	}
	conv.addBuiltinMacros()
	return conv, nil
}

func (conv *converter) Close() error {
	return os.RemoveAll(conv.WorkDir)
}

// Convert read the given LaTeX input file, converts the contents to
// EPUB format and writes the result to `book`.
func Convert(book *epub.Writer, inputFileName string) (err error) {
	conv, err := newConverter(book)
	if err != nil {
		return err
	}
	defer func() {
		e2 := conv.Close()
		if err == nil {
			err = e2
		}
	}()

	log.Println("tokenizing ...")
	err = conv.Tokenize(inputFileName)
	if err != nil {
		return err
	}

	log.Println("pass 1 ...")
	err = conv.Pass1()
	if err != nil {
		return err
	}

	log.Println("pass 2 ...")
	return conv.Pass2()
}
