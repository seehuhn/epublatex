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
	"html"
	"image"
	"image/draw"
	"log"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/seehuhn/epublatex/epub"
	"github.com/seehuhn/epublatex/latex/cache"
	"github.com/seehuhn/epublatex/latex/render"
)

var noCache = flag.Bool("latex-math-no-cache", false,
	"whether to disable the rendering cache")

const (
	renderRes = 3 * 96
	xHeight   = 4.30554 // x-height of cmi10 in TeX pt

	imgNames = "img%d.png"

	mathCachePruneLimit = 256 * 1024
)

type Renderer struct {
	book     epub.Writer
	preamble []string
	formulas map[string]int

	cache *cache.Cache
}

func NewRenderer(book epub.Writer) (*Renderer, error) {
	r := &Renderer{
		book:     book,
		formulas: make(map[string]int),
	}

	var err error
	r.cache, err = cache.NewCache("maths")
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) AddPreamble(line string) {
	r.preamble = append(r.preamble, line)
}

func (r *Renderer) AddFormula(env, formula string) {
	if strings.Contains(env, "%") {
		panic("invalid math environment " + env)
	}
	key := env + "%" + formula
	r.formulas[key]++
}

func (r *Renderer) Finish(out chan<- *render.BookImage) error {
	if len(r.formulas) == 0 {
		return nil
	}

	all := r.getFormulaInfo()

	needed := 0
	for _, info := range all {
		if info.Needed {
			needed++
		}
	}
	var c <-chan image.Image
	if needed > 0 {
		q, err := render.NewQueue(renderRes)
		if err != nil {
			return err
		}
		defer func() {
			e2 := q.Finish()
			if err == nil {
				err = e2
			}
		}()

		tmpl, err := template.New("tex").Parse(texTemplate)
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"Preamble": r.preamble,
			"Formulas": all,
		}
		c = q.Submit(tmpl, data)
	}

	err := r.gatherImages(all, c, out)
	if err != nil {
		return err
	}
	err = r.cache.Close(mathCachePruneLimit)
	return err
}

func (r *Renderer) getFormulaInfo() []*formulaInfo {
	var all []*formulaInfo
	for key, count := range r.formulas {
		parts := strings.SplitN(key, "%", 2)
		info := &formulaInfo{
			Key:     key,
			Env:     parts[0],
			Formula: parts[1],
			Count:   count,
			Needed:  *noCache || !r.cache.Has(key),
		}
		all = append(all, info)
	}
	sort.Sort(decreasingCount(all))
	for i, info := range all {
		info.FileName = strconv.Itoa(i)
	}
	return all
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
{{if .Needed -}}
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
{{end -}}
\end{document}
`

func (r *Renderer) gatherImages(all []*formulaInfo, in <-chan image.Image,
	out chan<- *render.BookImage) error {
	for _, info := range all {
		var crop func(imgIn image.Image) image.Image
		var cssClass string
		if info.Env == "$" {
			crop = cropInline
			cssClass = "imath"
		} else {
			crop = cropDisplayed
			cssClass = "dmath"
		}

		var img image.Image
		var err error
		if info.Needed {
			img = <-in
			if img == nil {
				log.Fatal("missing image")
			}
			img = crop(img)
			err = r.cache.Put(info.Key, img)
			if err != nil {
				return err
			}
		} else {
			img, err = r.cache.Get(info.Key)
			if err != nil {
				return err
			}
		}

		exWidth := float64(img.Bounds().Dx()) / float64(renderRes) * 72.27 / xHeight
		attr := fmt.Sprintf(` alt="%s" class="%s" style="width: %.2fex"`,
			html.EscapeString(info.Formula), cssClass, exWidth)

		job := &render.BookImage{
			Key:       info.Key,
			Image:     img,
			ImageAttr: attr,
			Name:      info.FileName,
			Folder:    "m",
			Type:      render.BookImageTypePNG,
		}
		out <- job
	}
	return nil
}

func cropInline(imgIn image.Image) image.Image {
	b := imgIn.Bounds()
	imgOut := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(imgOut, imgOut.Bounds(), imgIn, b.Min, draw.Src)

	// find the marker on the left
	y0 := 0
	for {
		if imgOut.Pix[y0*imgOut.Stride+3] != 0 {
			break
		}
		y0++
	}
	y1 := y0
	for {
		if imgOut.Pix[(y1+1)*imgOut.Stride+3] == 0 {
			break
		}
		y1++
	}
	yMid := (y0 + y1) / 2

	// find the width of the marker
	xMin := 0
	for imgOut.Pix[imgOut.PixOffset(xMin, yMid)+3] != 0 {
		xMin++
	}

	// find the top-most row of pixels used
	idx := 0
	for imgOut.Pix[idx+3] == 0 {
		idx += 4
	}
	yMin := idx / imgOut.Stride

	// find the bottom-most row of pixels used
	idx = imgOut.Rect.Max.Y*imgOut.Stride - 4
	for imgOut.Pix[idx+3] == 0 {
		idx -= 4
	}
	yMax := idx/imgOut.Stride + 1

	// Centre the crop window vertically.
	if y0-yMin > yMax-1-y1 {
		yMax = y0 + y1 - yMin + 1
	} else {
		yMin = y0 + y1 - yMax + 1
	}

	// crop left
leftLoop:
	for xMin < imgOut.Rect.Max.X {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMin, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftLoop
			}
		}
		xMin++
	}

	// crop right
	xMax := imgOut.Rect.Max.X
rightLoop:
	for xMax > xMin {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMax-1, y)
			if imgOut.Pix[idx+3] != 0 {
				break rightLoop
			}
		}
		xMax--
	}

	crop := image.Rectangle{
		Min: image.Point{xMin, yMin},
		Max: image.Point{xMax, yMax},
	}
	return imgOut.SubImage(crop)
}

func cropDisplayed(imgIn image.Image) image.Image {
	b := imgIn.Bounds()
	imgOut := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(imgOut, imgOut.Bounds(), imgIn, b.Min, draw.Src)

	// find the top-most row of pixels used
	idx := 0
	for imgOut.Pix[idx+3] == 0 {
		idx += 4
	}
	yMin := idx / imgOut.Stride

	// find the bottom-most row of pixels used
	idx = imgOut.Rect.Max.Y*imgOut.Stride - 4
	for imgOut.Pix[idx+3] == 0 {
		idx -= 4
	}
	yMax := idx/imgOut.Stride + 1

	// crop left and right
	xMin := 0
	xMax := imgOut.Rect.Max.X
leftRightLoop:
	for xMin < imgOut.Rect.Max.X {
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMin, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftRightLoop
			}
		}
		for y := yMin; y < yMax; y++ {
			idx := imgOut.PixOffset(xMax-1, y)
			if imgOut.Pix[idx+3] != 0 {
				break leftRightLoop
			}
		}
		xMin++
		xMax--
	}

	crop := image.Rectangle{
		Min: image.Point{xMin, yMin},
		Max: image.Point{xMax, yMax},
	}
	return imgOut.SubImage(crop)
}

type formulaInfo struct {
	Key string

	Env     string
	Formula string
	Count   int

	Needed   bool
	FileName string
}

type decreasingCount []*formulaInfo

func (dc decreasingCount) Len() int      { return len(dc) }
func (dc decreasingCount) Swap(i, j int) { dc[i], dc[j] = dc[j], dc[i] }
func (dc decreasingCount) Less(i, j int) bool {
	if dc[i].Count != dc[j].Count {
		return dc[i].Count > dc[j].Count
	}
	if dc[i].Formula != dc[j].Formula {
		return dc[i].Formula < dc[j].Formula
	}
	return dc[i].Env < dc[j].Env
}
