// render.go - rendering of TikZ images
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

package tikz

import (
	"flag"
	"fmt"
	"image"
	"log"
	"sync"
	"text/template"

	"golang.org/x/crypto/sha3"

	"github.com/seehuhn/epublatex/latex/cache"
	"github.com/seehuhn/epublatex/latex/render"
)

var noCache = flag.Bool("latex-tikz-no-cache", false,
	"whether to disable the TikZ rendering cache")

const (
	renderRes = 300     // render resolution [pixels / inch]
	exHeight  = 4.30554 // x-height of cmi10 [TeX pt / ex]
	exPerPix  = 72.27 / exHeight / float64(renderRes)

	tikzCachePruneLimit = 256 * 1024
)

type Renderer struct {
	out chan<- *render.BookImage

	preamble []string
	seen     map[string]bool
	cache    *cache.Cache

	queue    *render.Queue
	children *sync.WaitGroup

	tmpl *template.Template
}

func NewRenderer(out chan<- *render.BookImage) (*Renderer, error) {
	r := &Renderer{
		out:      out,
		seen:     make(map[string]bool),
		children: &sync.WaitGroup{},
	}

	queue, err := render.NewQueue(renderRes)
	if err != nil {
		return nil, err
	}
	r.queue = queue

	cache, err := cache.NewCache("tikz")
	if err != nil {
		return nil, err
	}
	r.cache = cache

	tmpl, err := template.New("tikz").Parse(tikzTemplate)
	if err != nil {
		return nil, err
	}
	r.tmpl = tmpl

	return r, nil
}

func (r *Renderer) Finish() error {
	r.children.Wait()
	err := r.queue.Finish()
	e2 := r.cache.Close(tikzCachePruneLimit)
	if err == nil {
		err = e2
	}

	return err
}

func (r *Renderer) AddPreamble(line string) {
	r.preamble = append(r.preamble, line)
}

func (r *Renderer) AddPicture(picture string) {
	key := r.makeKey(picture)
	if r.seen[key] {
		// avoid including the same image twice
		return
	}
	r.seen[key] = true

	info := &pictureInfo{
		key:     key,
		picture: picture,
	}

	if !*noCache && r.cache.Has(key) {
		img, err := r.cache.Get(key)
		if err != nil {
			log.Println("cache failure:", err)
			goto render
		}
		r.submit(info, img)
		return
	}

render:
	data := map[string]interface{}{
		"Preamble": r.preamble,
		"Body":     picture,
	}
	in := r.queue.Submit(r.tmpl, data)

	r.children.Add(1)
	go func(info *pictureInfo) {
		img := <-in
		if img == nil {
			log.Println("missing image", info.picture)
		} else {
			r.cache.Put(info.key, img)
			r.submit(info, img)
		}
		r.children.Done()

		for range in {
			log.Println("error: received unexpected image from renderer")
		}
	}(info)
}

func (r *Renderer) submit(info *pictureInfo, img image.Image) {
	alt := "[image]"
	if len(info.picture) <= 60 {
		// TODO(voss): is this case really relevant?  Can we get a
		// user-supplied alt string?
		alt = info.picture
	}
	exWidth := float64(img.Bounds().Dx()) * exPerPix
	style := fmt.Sprintf("width: %.2fex", exWidth)
	job := &render.BookImage{
		Env:  "tikzpicture",
		Body: info.picture,

		Alt:      alt,
		CssClass: "tikzpicture",
		Style:    style,

		Image: img,
		Type:  render.BookImageTypePNG,
	}
	r.out <- job
}

func (r *Renderer) makeKey(picture string) string {
	// TODO(voss): should the preamble affect the key?
	hash := sha3.Sum224([]byte(picture))
	return fmt.Sprintf("tikz:%d:%f:%x", renderRes, exHeight, hash)
}

type pictureInfo struct {
	key     string
	picture string
}

const tikzTemplate = `\documentclass[tikz]{standalone}
{{range .Preamble -}}
{{.}}
{{end}}
\begin{document}
\begin{tikzpicture}
{{.Body}}
\end{tikzpicture}
\end{document}
`
