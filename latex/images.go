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
	"encoding/base64"
	"html"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"strings"

	"golang.org/x/crypto/sha3"

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

		// try to make a stable name
		h := sha3.NewShake128()
		h.Write([]byte(job.Env))
		h.Write([]byte{0})
		h.Write([]byte(job.Body))
		buf := make([]byte, 5)
		h.Read(buf)
		rawName := base64.RawURLEncoding.EncodeToString(buf)

		file := conv.Book.RegisterFile(rawName, mime, false)
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

		attrs := []string{
			` src="` + html.EscapeString(file.Path) + `"`,
		}
		if job.CssClass != "" {
			attrs = append(attrs,
				` class="`+html.EscapeString(job.CssClass)+`"`)
		}
		if job.Alt != "" {
			attrs = append(attrs,
				` alt="`+html.EscapeString(job.Alt)+`"`)
		}
		if job.Style != "" {
			attrs = append(attrs,
				` style="`+job.Style+`"`)
		}
		key := job.Env + "%" + job.Body
		conv.Images[key] = `<img` + strings.Join(attrs, "") + `/>`
	}
	res <- firstError
}
