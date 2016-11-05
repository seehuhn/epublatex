package render

import (
	"testing"
	"text/template"
)

const texTemplate = `\documentclass{minimal}

\usepackage[paperwidth=6in,paperheight=9in,margin=0pt]{geometry}

\usepackage{pdfrender}
\pdfrender{TextRenderingMode=2,LineWidth=0.05pt}

\begin{document}
{{.}}
\end{document}
`

func TestQueue(t *testing.T) {
	queue, err := NewQueue(150)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Finish()

	tmpl := template.Must(template.New("tex").Parse(texTemplate))
	c := queue.Submit(tmpl, "Hello world!\n\\newpage\ngood bye\n")

	count := 0
	for img := range c {
		count++
		if img == nil {
			t.Errorf("image %d is missing", count)
		}
	}
	if count != 2 {
		t.Error("wrong number of pages, expected 2, got", count)
	}
}
