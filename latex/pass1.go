// pass1.go -
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
	"io"
	"os"

	"github.com/seehuhn/ebook/latex/math"
	"github.com/seehuhn/ebook/latex/tokenizer"
)

// Pass1 renders all formulas as images and extracts the cross-references.
func (conv *converter) Pass1() error {
	var labels []*xRef
	ref := -1
	refType := ""
	refName := ""

	renderer, err := math.NewRenderer(conv.Book)
	if err != nil {
		return err
	}
	// TODO(voss): handle this properly
	renderer.AddPreamble("\\usepackage{amsfonts}")
	renderer.AddPreamble("\\usepackage{amsmath}")
	renderer.AddPreamble("\\DeclareMathOperator*{\\argmax}{arg\\,max}")
	mathMode := 0
	mathEnv := ""
	var mathTokens tokenizer.TokenList

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
		if mathMode == 0 {
			switch {
			case token.Type == tokenizer.TokenOther && token.Name == "$":
				mathMode = 1
				mathEnv = token.Name
			case token.Type == tokenizer.TokenOther && token.Name == "$$":
				mathMode = 2
				mathEnv = token.Name
			case token.Type == tokenizer.TokenMacro && token.Name == "\\begin":
				env := token.Args[0].String()
				if env == "equation" || env == "equation*" {
					mathMode = 2
					mathEnv = env
				}
			}
			if mathMode != 0 {
				goto NextToken
			}
		} else {
			eom := false
			switch {
			case token.Type == tokenizer.TokenOther && token.Name == mathEnv:
				eom = true
			case token.Type == tokenizer.TokenMacro && token.Name == "\\end":
				env := token.Args[0].String()
				if env == mathEnv {
					eom = true
				}
			}

			if eom {
				body := mathTokens.FormatMaths()
				if mathMode == 1 {
					renderer.AddInline(body)
				} else {
					renderer.AddDisplayed(body)
				}
				mathMode = 0
				mathTokens = nil
			} else {
				mathTokens = append(mathTokens, token)
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
					m.HTMLOutput(token.Args, conv)
				}
			}
		}

	NextToken:
		pos++
	}

	images, err := renderer.Finish()
	if err != nil {
		return err
	}
	conv.Images = images
	conv.Labels = labels
	return nil
}
