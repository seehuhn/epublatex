// images.go -
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
	"fmt"
	"html"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"

	"github.com/seehuhn/epublatex/latex/render"
)

func (conv *converter) GetImage(env, body string) string {
	key := env + "%" + body
	res, ok := conv.Images[key]
	if !ok {
		log.Printf("missing image for body %q", body)
	}
	return res
}

// imageAdder serialises the process of adding new image files to the book.
func (conv *converter) imageAdder(in <-chan *render.BookImage, res chan<- error) {
	var firstError error
	for job := range in {
		var mime string
		var enc func(w io.Writer, m image.Image) error
		switch job.Type {
		case render.BookImageTypePNG:
			mime = "image/png"
			enc = png.Encode
		default:
			mime = "image/jpeg"
			enc = func(w io.Writer, m image.Image) error {
				return jpeg.Encode(w, m, nil)
			}
		}

		name := job.Name
		if job.Folder != "" {
			name = job.Folder + "/" + name
		}
		file := conv.Book.RegisterFile(name, mime, false)
		w, err := conv.Book.CreateFile(file)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			continue
		}
		err = enc(w, job.Image)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			continue
		}

		s := fmt.Sprintf(`<img src="%s"%s/>`,
			html.EscapeString(file.Path), job.ImageAttr)
		conv.Images[job.Key] = s
	}
	res <- firstError
}
