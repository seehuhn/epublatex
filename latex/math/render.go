// render.go -
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

package math

import (
	"flag"
	"fmt"
	"image"
	"log"
	"strings"
	"sync"
	"text/template"

	"github.com/seehuhn/epublatex/latex/cache"
	"github.com/seehuhn/epublatex/latex/render"
)

var noCache = flag.Bool("latex-math-no-cache", false,
	"whether to disable the rendering cache")

const (
	renderRes = 3 * 96  // render resolution [pixels / inch]
	exHeight  = 4.30554 // x-height of cmi10 [TeX pt / ex]
	exPerPix  = 72.27 / exHeight / float64(renderRes)

	imgNames = "img%d.png"

	batchSize           = 10
	mathCachePruneLimit = 256 * 1024
)

type Renderer struct {
	out chan<- *render.BookImage

	preamble []string
	seen     map[string]bool
	cache    *cache.Cache

	batch    []*formulaInfo
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

	cache, err := cache.NewCache("maths")
	if err != nil {
		return nil, err
	}
	r.cache = cache

	tmpl, err := template.New("tex").Parse(texTemplate)
	if err != nil {
		return nil, err
	}
	r.tmpl = tmpl

	return r, nil
}

func (r *Renderer) Finish() error {
	if len(r.batch) > 0 {
		r.runBatch()
	}
	r.children.Wait()

	err := r.queue.Finish()

	e2 := r.cache.Close(mathCachePruneLimit)
	if err == nil {
		err = e2
	}

	return err
}

func (r *Renderer) AddPreamble(line string) {
	r.preamble = append(r.preamble, line)
}

func (r *Renderer) AddFormula(env, formula string) {
	if strings.Contains(env, "%") {
		panic("invalid math environment " + env)
	}
	key := r.makeKey(env, formula)
	if r.seen[key] {
		// avoid including the same image twice
		return
	}
	r.seen[key] = true

	info := &formulaInfo{
		key:     key,
		Env:     env,
		Formula: formula,
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
	r.batch = append(r.batch, info)
	if len(r.batch) >= batchSize {
		r.runBatch()
	}
}

func (r *Renderer) runBatch() {
	all := r.batch
	r.batch = nil

	data := map[string]interface{}{
		"Preamble": r.preamble,
		"Formulas": all,
	}
	in := r.queue.Submit(r.tmpl, data)

	r.children.Add(1)
	go func() {
		for _, info := range all {
			img := <-in
			if img == nil {
				log.Println("missing image", info.Env, info.Formula)
				continue
			}
			if info.Env == "$" {
				img = cropInline(img)
			} else {
				img = cropDisplayed(img)
			}

			r.cache.Put(info.key, img)
			r.submit(info, img)
		}
		r.children.Done()

		for range in {
			log.Println("error: unexpected image received from renderer")
		}
	}()
}

func (r *Renderer) submit(info *formulaInfo, img image.Image) {
	alt := "[formula]"
	if len(info.Formula) <= 60 {
		alt = info.Formula
	}
	var cssClass string
	if info.Env == "$" {
		cssClass = "imath"
	} else {
		cssClass = "dmath"
	}
	exWidth := float64(img.Bounds().Dx()) * exPerPix
	style := fmt.Sprintf("width: %.2fex", exWidth)
	job := &render.BookImage{
		Env:  info.Env,
		Body: info.Formula,

		Alt:      alt,
		CssClass: cssClass,
		Style:    style,

		Image: img,
		Type:  render.BookImageTypePNG,
	}
	r.out <- job
}

func (r *Renderer) makeKey(env, formula string) string {
	// TODO(voss): should the preamble affect the key?
	// TODO(voss): would hashing be beneficial?
	return fmt.Sprintf("%d%%%f%%%s%%%s", renderRes, exHeight, env, formula)
}

type formulaInfo struct {
	key     string
	Env     string
	Formula string
}

const texTemplate = `\documentclass{minimal}

\usepackage[paperwidth=6in,paperheight=9in,margin=0pt]{geometry}

\usepackage{pdfrender}
\pdfrender{TextRenderingMode=2,LineWidth=0.05pt}

{{range .Preamble -}}
{{.}}
{{end}}
\parindent0pt
\parskip0pt

\begin{document}
\fontsize{10}{12}\selectfont

{{range .Formulas -}}
{{if eq .Env "$" -}}
\vrule width6bp height4.3pt depth0pt \kern6bp
${{.Formula}}$
{{else -}}
\begin{ {{- .Env -}} }
  {{.Formula}}
\end{ {{- .Env -}} }
{{end -}}
\newpage

{{end -}}
\end{document}
`
