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
	"encoding/hex"
	"flag"
	"fmt"
	"html"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/seehuhn/epublatex/epub"
	"golang.org/x/crypto/sha3"
)

const (
	renderRes = 3 * 96
	imgNames  = "img%d.png"
	xHeight   = 4.30554 // x-height of cmi10 in TeX pt
)

var cacheDir = flag.String("math-cache", "",
	"cache directory for maths rendering")

type Renderer struct {
	book      *epub.Writer
	preamble  []string
	inline    map[string]bool
	displayed map[string]bool

	cacheDir string
}

func NewRenderer(book *epub.Writer) (*Renderer, error) {
	r := &Renderer{
		book:      book,
		inline:    make(map[string]bool),
		displayed: make(map[string]bool),
	}

	cacheDir := *cacheDir
	if len(r.cacheDir) == 0 {
		cacheDir = os.Getenv("JV_EBOOK_CACHE")
	}
	if len(cacheDir) == 0 {
		cacheDir = os.ExpandEnv(defaultCacheDir)
		cacheDir = filepath.Join(cacheDir, "de.seehuhn.ebook")
	}

	r.cacheDir = filepath.Join(cacheDir, "maths")
	err := os.MkdirAll(r.cacheDir, 0755)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Renderer) cacheFileName(class, formula string) string {
	h := sha3.NewShake128()
	h.Write([]byte(class + ":" + strconv.Itoa(renderRes) + ":" + formula))
	buf := make([]byte, 16)
	h.Read(buf)
	fileName := hex.EncodeToString(buf) + ".png"
	return filepath.Join(r.cacheDir, fileName)
}

func (r *Renderer) isCached(class, formula string) bool {
	filePath := r.cacheFileName(class, formula)
	_, err := os.Stat(filePath)
	return err == nil
}

func (r *Renderer) writeCached(class, formula string, img image.Image) error {
	filePath := r.cacheFileName(class, formula)
	fd, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fd.Close()

	return png.Encode(fd, img)
}

func (r *Renderer) loadCached(class, formula string) (image.Image, error) {
	filePath := r.cacheFileName(class, formula)
	return readImage(filePath)
}

func (r *Renderer) AddPreamble(line string) {
	r.preamble = append(r.preamble, line)
}

func (r *Renderer) AddInline(formula string) {
	r.inline[formula] = true
}

func (r *Renderer) AddDisplayed(formula string) {
	r.displayed[formula] = true
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
{{range .Inline}}
\vrule width6bp height4.3pt depth0pt \kern6bp ${{.}}$
\newpage
{{end}}
{{range .Displayed}}
\begin{equation*}
  {{.}}
\end{equation*}
\newpage
{{end}}
\end{document}
`

func (r *Renderer) writeTexFile(name string, inline, displayed []string) (err error) {
	texFile, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		e1 := texFile.Close()
		if err == nil {
			err = e1
		}
	}()

	tmpl, err := template.New("tex").Parse(texTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(texFile, map[string][]string{
		"Preamble":  r.preamble,
		"Inline":    inline,
		"Displayed": displayed,
	})
}

func readImage(fname string) (image.Image, error) {
	fd, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, err
	}
	return img, nil
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

func (r *Renderer) registerResults(baseIdx int, list []string,
	cssClass string, crop func(image.Image) image.Image,
	res map[string]string, workDir string) error {
	for i, formula := range list {
		k := i + baseIdx
		imageFileName := filepath.Join(workDir, fmt.Sprintf(imgNames, k))
		img, err := readImage(imageFileName)
		if err != nil {
			return err
		}
		img = crop(img)
		err = r.writeCached(cssClass, formula, img)
		if err != nil {
			return err
		}

		name := strconv.Itoa(k)
		file := r.book.RegisterFile("m/"+name, "image/png", false)
		w, err := r.book.CreateFile(file)
		if err != nil {
			return err
		}
		err = png.Encode(w, img)
		if err != nil {
			return err
		}

		exWidth := float64(img.Bounds().Dx()) / float64(renderRes) * 72.27 / xHeight
		s := fmt.Sprintf(
			`<img alt="%s" src="%s" class="%s" style="width: %.2fex"/>`,
			html.EscapeString(formula), html.EscapeString(file.Path),
			cssClass, exWidth)
		res[formula] = s
	}
	return nil
}

func (r *Renderer) registerCached(baseIdx int, list []string,
	cssClass string, res map[string]string) error {
	for i, formula := range list {
		k := i + baseIdx
		img, err := r.loadCached(cssClass, formula)
		if err != nil {
			return err
		}

		name := strconv.Itoa(k)
		file := r.book.RegisterFile("m/"+name, "image/png", false)
		w, err := r.book.CreateFile(file)
		if err != nil {
			return err
		}
		err = png.Encode(w, img)
		if err != nil {
			return err
		}

		exWidth := float64(img.Bounds().Dx()) / float64(renderRes) * 72.27 / xHeight
		s := fmt.Sprintf(
			`<img alt="%s" src="%s" class="%s" style="width: %.2fex"/>`,
			html.EscapeString(formula), html.EscapeString(file.Path),
			cssClass, exWidth)
		res[formula] = s
	}
	return nil
}

func (r *Renderer) Finish() (res *Images, err error) {
	res = &Images{
		inline:    make(map[string]string),
		displayed: make(map[string]string),
	}
	if len(r.inline) == 0 && len(r.displayed) == 0 {
		return res, nil
	}

	var inline []string
	var inlineCached []string
	for formula := range r.inline {
		if r.isCached("imath", formula) {
			inlineCached = append(inlineCached, formula)
		} else {
			inline = append(inline, formula)
		}
	}
	var displayed []string
	var displayedCached []string
	for formula := range r.displayed {
		if r.isCached("dmath", formula) {
			displayedCached = append(displayedCached, formula)
		} else {
			displayed = append(displayed, formula)
		}
	}

	if len(inline)+len(displayed) > 0 {
		workDir, err := ioutil.TempDir("", "epub")
		if err != nil {
			return nil, err
		}
		defer func() {
			e2 := os.RemoveAll(workDir)
			if err == nil {
				err = e2
			}
		}()

		oldDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		err = os.Chdir(workDir)
		if err != nil {
			return nil, err
		}
		defer func() {
			e2 := os.Chdir(oldDir)
			if err == nil {
				err = e2
			}
		}()

		texFileName := filepath.Join(workDir, "all.tex")
		err = r.writeTexFile(texFileName, inline, displayed)
		if err != nil {
			return nil, err
		}

		ltx := exec.Command("pdflatex", "-interaction=nonstopmode", texFileName)
		output, err := ltx.Output()
		if err != nil {
			if e2, ok := err.(*exec.ExitError); ok {
				log.Println("Rendering formulas using LaTeX failed:", e2)
				log.Println("--- begin LaTeX output ---")
				log.Println(string(output))
				log.Println("--- end LaTeX output ---")
			}
			return nil, err
		}

		pdfFileName := filepath.Join(workDir, "all.pdf")
		resolution := strconv.Itoa(renderRes)
		gs := exec.Command("gs", "-dSAFER", "-dBATCH", "-dNOPAUSE", "-r"+resolution,
			"-sDEVICE=pngalpha", "-dTextAlphaBits=4", "-sOutputFile="+imgNames,
			pdfFileName)
		output, err = gs.Output()
		if err != nil {
			if e2, ok := err.(*exec.ExitError); ok {
				log.Println("Converting formulas to .png using gs failed:", e2)
				log.Println("--- begin gs output ---")
				log.Println(string(output))
				log.Println("--- end gs output ---")
			}
			return nil, err
		}

		err = r.registerResults(1, inline, "imath", cropInline,
			res.inline, workDir)
		if err != nil {
			return nil, err
		}
		err = r.registerResults(1+len(inline), displayed, "dmath", cropDisplayed,
			res.displayed, workDir)
		if err != nil {
			return nil, err
		}
	}
	err = r.registerCached(1+len(inline)+len(displayed), inlineCached, "imath",
		res.inline)
	if err != nil {
		return nil, err
	}
	err = r.registerCached(1+len(inline)+len(displayed)+len(inlineCached),
		displayedCached, "dmath", res.displayed)
	if err != nil {
		return nil, err
	}

	return res, nil
}
