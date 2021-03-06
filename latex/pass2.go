// pass2.go - generate output
//
// Copyright (C) 2016, 2017  Jochen Voss <voss@seehuhn.de>
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
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/seehuhn/epublatex/latex/tokenizer"
)

func (conv *converter) convertHTML(tokens tokenizer.TokenList) string {
	var res []string
	inMath := false
	var mathTokens tokenizer.TokenList
	for _, token := range tokens {
		switch {
		case token.Type == tokenizer.TokenOther && token.Name == "$" && !inMath:
			inMath = true
		case token.Type == tokenizer.TokenOther && token.Name == "$" && inMath:
			inMath = false
			body := mathTokens.FormatMaths()
			res = append(res, conv.GetImage("$", body))
			mathTokens = nil
		case inMath:
			mathTokens = append(mathTokens, token)
		case token.Type == tokenizer.TokenMacro:
			if m, ok := conv.Macros[token.Name]; ok {
				res = append(res, m.HTMLOutput(token.Args, conv))
			} else {
				log.Printf("unknown macro %q", token.Name)
			}
		case token.Type == tokenizer.TokenSpace:
			res = append(res, " ")
		case token.Type == tokenizer.TokenWord:
			res = append(res, token.Name)
		case token.Type == tokenizer.TokenOther:
			switch token.Name {
			case "~":
				res = append(res, noBreakSpace)
			case "``":
				res = append(res, "<q>")
			case "''":
				res = append(res, "</q>")
			default:
				res = append(res, token.Name)
			}
		}
	}
	return strings.Join(res, "")
}

// Pass2 converts the text to HTML.
func (conv *converter) Pass2() (err error) {
	var mathMode isEnd
	var mathEnv string
	var mathTokens tokenizer.TokenList
	var mathLabel string

	w := newWriter(conv.Book, conv.SourceDir)
	defer func() {
		e2 := w.Flush()
		if err == nil {
			err = e2
		}
	}()

	// The following loop must match the corresponding code in
	// the .Pass1() method.
	conv.Section = nil
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
			if mathMode(token) {
				if mathLabel != "" {
					var id, name string
					for _, label := range conv.Labels {
						if label.Label == mathLabel {
							id = label.ID
							name = label.Name
						}
					}
					w.WriteString("<br/>")
					w.WriteString(`<span class="latex-eqno" id="` + id +
						`">(` + name + `)</span>`)
				}
				w.WriteString(conv.GetImage(mathEnv, mathTokens.FormatMaths()))

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
			goto NextToken
		}

		switch {
		case token.Type == tokenizer.TokenMacro:
			switch token.Name {
			case "\\epubcover":
				fileName := token.Args[0].String()
				log.Println("EPUB cover", fileName)
				err = w.AddCoverImage(fileName)
				if err != nil {
					return err
				}
			case "\\epubmaketitle":
				err = w.WriteTitle(conv.Title, conv.Author)
				if err != nil {
					return err
				}
			case "\\epubsection":
				conv.Section.Inc(1)
				conv.resetCounters(1, "section")
				title := conv.convertHTML(token.Args[1].Value)
				id := conv.xRefLookup(pos)
				err := w.AddSection(1, title, id)
				if err != nil {
					return err
				}
			case "\\epubsubsection":
				conv.Section.Inc(2)
				conv.resetCounters(2, "subsection")
				title := conv.convertHTML(token.Args[1].Value)
				id := conv.xRefLookup(pos)
				err := w.AddSection(2, title, id)
				if err != nil {
					return err
				}
			case "%tikz%":
				picture := token.Args[1].String()
				w.WriteString(conv.GetImage("tikzpicture", picture))

			case "\\begin":
				name := token.Args[0].String()

				id := conv.xRefLookup(pos)
				if id == "" {
					id = "pos-" + strconv.Itoa(pos)
				}

				var classes []string
				var pfx string
				if env, ok := conv.Envs[name]; ok {
					classes = env.CSSClasses
					ctr := conv.Counters[env.Counter]
					ctr.Value++
					pfx = env.Prefix
					pfx = "<b>" + pfx + noBreakSpace + ctr.String() + ".</b>"
				}

				if len(conv.EnvStack) > 0 {
					err := w.StartBlock(name, classes, id)
					if err == nil && pfx != "" {
						w.WriteString(pfx)
						err = w.EndWord()
					}
					if err != nil {
						return err
					}
				}

				conv.EnvStack = append(conv.EnvStack, name)
			case "\\end":
				name := token.Args[0].String()
				n := len(conv.EnvStack)
				if n > 0 && conv.EnvStack[n-1] == name {
					conv.EnvStack = conv.EnvStack[:n-1]
				} else {
					pos := -1
					for i, env := range conv.EnvStack {
						if env == name {
							pos = i
							break
						}
					}
					if pos < 0 {
						log.Println("environment", name, "was not open")
					} else {
						log.Println("environment", conv.EnvStack[n-1],
							"was not closed")
						conv.EnvStack = conv.EnvStack[:pos]
					}
				}

				if len(conv.EnvStack) > 0 {
					err := w.EndBlock()
					if err != nil {
						return err
					}
				}
			case "\\bf":
				if !w.state.isBold {
					w.WriteString("<b>")
					w.state.isBold = true
				}
			case "\\it":
				if !w.state.isItalic {
					w.WriteString("<i>")
					w.state.isItalic = true
				}

			default:
				// TODO(voss): add a more general mechanism to select
				// vertical mode.
				verticalMode := token.Name == "%verbatim%"
				s := conv.convertHTML(tokenizer.TokenList{token})
				if verticalMode {
					err := w.WriteVertical(s)
					if err != nil {
						return err
					}
				} else {
					w.WriteString(s)
				}
			}
		case token.Type == tokenizer.TokenWord:
			w.WriteString(token.Name)
		case token.Type == tokenizer.TokenOther && token.Name == "{":
			stateCopy := *w.state
			w.stack = append(w.stack, &stateCopy)
		case token.Type == tokenizer.TokenOther && token.Name == "}":
			oldState := w.state
			n := len(w.stack)
			w.state, w.stack = w.stack[n-1], w.stack[:n-1]
			// TODO(voss): avoid <b><i></b></i>
			if w.state.isBold != oldState.isBold {
				if w.state.isBold {
					w.WriteString("<b>")
				} else {
					w.WriteString("</b>")
				}
			}
			if w.state.isItalic != oldState.isItalic {
				if w.state.isItalic {
					w.WriteString("<i>")
				} else {
					w.WriteString("</i>")
				}
			}
		case token.Type == tokenizer.TokenOther:
			w.WriteString(conv.convertHTML(tokenizer.TokenList{token}))

		case len(conv.EnvStack) == 0:
			// Don't try to write space outside the {document} environment.

		case token.Type == tokenizer.TokenSpace:
			err = w.EndWord()
			if err != nil {
				return err
			}
		case token.Type == tokenizer.TokenEmptyLine:
			err = w.EndParagraph()
			if err != nil {
				return err
			}
		}

	NextToken:
		pos++
	}
	return nil
}
