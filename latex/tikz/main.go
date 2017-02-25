// +build ignore

package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/seehuhn/epublatex/latex/render"
	"github.com/seehuhn/epublatex/latex/tikz"
)

const picture = `  \fill[domain=-4.5:-1.96,smooth,samples=11,color=gray] plot (\x,{2.5*exp(-\x*\x/2)}) -- (-1.96,0) -- (-4,0);
  \fill[domain=1.96:4.5,smooth,samples=11,color=gray] plot (\x,{2.5*exp(-\x*\x/2)}) -- (4.5,0) -- (1.96,0);
  \draw[->] (-5,0) -- (5,0) node[below]{$x$};
  \draw[->] (0,-0.5) -- (0, 3) node[right]{$\phi_{0,1}(x)$};
  \draw[domain=-4.5:4.5,smooth,samples=21] plot (\x,{2.5*exp(-\x*\x/2)});
  \draw (1.96, 0.3662243) -- (1.96, -0.1) node[below]{$q_{1-\alpha/2}$};
  \draw (-1.96, 0.3662243) -- (-1.96, -0.1) node[below]{$-q_{1-\alpha/2}$};
  \draw[decorate,decoration={brace}] (5, -0.8) -- node[below,yshift=-3pt]{prob.~$\alpha/2$} (1.98,-0.8);
  \draw[decorate,decoration={brace}] (1.94, -0.8) -- node[below,yshift=-3pt]{prob.~$1 - \alpha$} (-1.94,-0.8);
  \draw[decorate,decoration={brace}] (-1.98, -0.8) -- node[below,yshift=-3pt]{prob.~$\alpha/2$} (-5,-0.8);`

func main() {
	flag.Parse()

	out := make(chan *render.BookImage)
	renderer, err := tikz.NewRenderer(out)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	go func(in <-chan *render.BookImage) {
		count := 0
		for img := range in {
			count++

			outName := fmt.Sprintf("img%02d.png", count)
			out, err := os.Create(outName)
			if err != nil {
				log.Fatal(err)
			}
			defer out.Close()
			err = png.Encode(out, img.Image)
			if err != nil {
				log.Fatal(err)
			}
		}
		close(done)
	}(out)

	renderer.AddPreamble(`\usetikzlibrary{decorations.pathreplacing}`)
	renderer.AddPicture(picture)

	err = renderer.Finish()
	if err != nil {
		log.Fatal(err)
	}

	close(out)
	<-done
}
