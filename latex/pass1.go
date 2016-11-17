// pass1.go - render formulas/images, extract cross references
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
	"encoding/gob"
	"errors"
	"io"
	"log"
	"os"

	"github.com/seehuhn/epublatex/latex/math"
	"github.com/seehuhn/epublatex/latex/render"
	"github.com/seehuhn/epublatex/latex/tokenizer"
)

// ErrUnterminatedMath indicates the \end{...} tag for a LaTeX maths
// environment was not found.
var ErrUnterminatedMath = errors.New("maths environment not terminated")

// Pass1 renders all formulas and tikz images, and extracts the
// cross-references.
func (conv *converter) Pass1() error {
	conv.Images = make(map[string]string)
	imageChan := make(chan *render.BookImage)
	resChan := make(chan error)
	go conv.imageAdder(imageChan, resChan)

	var labels []*xRef
	ref := -1
	refType := ""
	refName := ""

	renderer, err := math.NewRenderer(imageChan)
	if err != nil {
		return err
	}
	// TODO(voss): handle this properly
	renderer.AddPreamble("\\usepackage{amsfonts}")
	renderer.AddPreamble("\\usepackage{amsmath}")
	renderer.AddPreamble("\\DeclareMathOperator*{\\argmax}{arg\\,max}")
	var mathMode isEnd
	var mathEnv string
	var mathTokens tokenizer.TokenList
	var mathLabel string

	// The following loop must match the corresponding code in
	// the .Pass2() method.
	tokFile, err := os.Open(conv.TokenFileName)
	if err != nil {
		return err
	}
	defer tokFile.Close()
	toks := gob.NewDecoder(tokFile)
	pos := 0
	for {
		var token *tokenizer.Token
		err := toks.Decode(&token)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// maths formulas
		if mathMode == nil {
			mathEnv, mathMode = conv.IsMathStart(token)
			if mathMode != nil {
				mathLabel = ""
				goto NextToken
			}
		} else {
			// we only need to check this in once, in pass 1
			if token.Type == tokenizer.TokenEmptyLine {
				log.Println("maths environment not terminated\n" +
					mathTokens.FormatMaths())
				return ErrUnterminatedMath
			}

			if mathMode(token) {
				renderer.AddFormula(mathEnv, mathTokens.FormatMaths())

				mathMode = nil
				mathTokens = nil
			} else {
				ignore := false
				if token.Type == tokenizer.TokenMacro &&
					token.Name == "\\label" &&
					mathLabel == "" {
					mathLabel = token.Args[0].String()
					ignore = true
				}
				if !ignore {
					mathTokens = append(mathTokens, token)
				}
			}
		}

		// handle cross-references
		if token.Type == tokenizer.TokenMacro {
			switch token.Name {
			case "\\epubsection":
				conv.Section.Inc(1)
				conv.resetCounters(1, "section")
				ref = pos
				refType = "Section"
				refName = conv.Section.String()
			case "\\epubsubsection":
				conv.Section.Inc(2)
				conv.resetCounters(2, "subsection")
				ref = pos
				refType = "Subsection"
				refName = conv.Section.String()
			case "\\begin":
				name := token.Args[0].String()
				if env, ok := conv.Envs[name]; ok {
					ref = pos
					refType = env.Prefix
					refName = conv.Counters[env.Counter].Inc()
				}
			case "\\label":
				label := token.Args[0].String()
				target := &xRef{
					Label:   label,
					Chapter: conv.Section[0],
					ID:      xRefNormalise(label, labels),
					Pos:     ref,
					Type:    refType,
					Name:    refName,
				}
				labels = append(labels, target)

			default:
				m, ok := conv.Macros[token.Name]
				if ok {
					// run for side-effects only, discard output
					_ = m.HTMLOutput(token.Args, conv)
				}
			}
		}

	NextToken:
		pos++
	}

	err = renderer.Finish()
	close(imageChan)
	if err != nil {
		return err
	}

	err = <-resChan
	if err != nil {
		return err
	}

	conv.Labels = labels
	return nil
}
