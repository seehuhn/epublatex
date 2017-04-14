// main.go -
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

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/seehuhn/epublatex/epub"
	"github.com/seehuhn/epublatex/latex"
)

var output = flag.String("output", "", "the output file name")
var html = flag.Bool("html", false, "generate HTML instead of EPUB")

func main() {
	log.Println("start")
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatal("usage: main <input.tex>")
	}
	inputName := flag.Arg(0)

	outputName := *output
	if outputName == "" {
		base := strings.TrimSuffix(filepath.Base(inputName), ".tex")
		outputName = base
		if !*html {
			outputName = outputName + ".epub"
		}
	}
	log.Println("writing", outputName)

	BookID := "my second ebook (test)"

	var book *epub.Book
	var err error
	if *html {
		book, err = epub.NewXhtmlWriter(outputName, BookID)
	} else {
		out, err := os.Create(outputName)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
		book, err = epub.NewEpubWriter(out, BookID)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer func() {
		err := book.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = latex.Convert(book, inputName)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}
