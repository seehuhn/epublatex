// pass2.go -
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
			res = append(res, conv.Images.GetInline(body))
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
	mathMode := 0
	mathEnv := ""
	var mathTokens tokenizer.TokenList
	var mathLabel string

	w := newWriter(conv.Book, conv.SourceDir)
	defer func() {
		e2 := w.Flush()
		if err == nil {
			err = e2
		}
	}()

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
				mathLabel = ""
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
				var extra, img string
				if mathMode == 1 {
					img = conv.Images.GetInline(body)
				} else {
					img = conv.Images.GetDisplayed(body)
					if mathLabel != "" {
						var id, name string
						for _, label := range conv.Labels {
							if label.Label == mathLabel {
								id = label.ID
								name = label.Name
							}
						}
						extra = `<br/><span class="latex-eqno" id="` + id +
							`">(` + name + `)</span>`
					}
				}
				w.WriteString(extra + img)

				mathMode = 0
				mathTokens = nil
			} else {
				if token.Type == tokenizer.TokenMacro &&
					token.Name == "\\label" &&
					mathLabel == "" {
					mathLabel = token.Args[0].String()
				}
				mathTokens = append(mathTokens, token)
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

			default:
				s := conv.convertHTML(tokenizer.TokenList{token})
				w.WriteString(s)
			}
		case token.Type == tokenizer.TokenWord:
			w.WriteString(token.Name)
		case token.Type == tokenizer.TokenOther:
			w.WriteString(conv.convertHTML(tokenizer.TokenList{token}))

		case len(conv.EnvStack) == 0:
			// Don't try to write space outside the {document} environment.

		case token.Type == tokenizer.TokenSpace:
			w.EndWord()
		case token.Type == tokenizer.TokenEmptyLine:
			w.EndParagraph()

		}

	NextToken:
		pos++
	}
	return nil
}
