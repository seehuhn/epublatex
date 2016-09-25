// tokenizer.go -
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

package tokenizer

import (
	"bytes"
	"log"

	"github.com/seehuhn/epublatex/latex/scanner"
)

// A Tokenizer can be used to split a LaTeX file into syntactic units.
// User-defined macros are expanded in the process.
type Tokenizer struct {
	scanner.Scanner

	macros       map[string]macro
	environments map[string]environment
}

// NewTokenizer creates and initialises a new Tokenizer.
func NewTokenizer() *Tokenizer {
	p := &Tokenizer{
		macros:       make(map[string]macro),
		environments: make(map[string]environment),
	}
	p.addBuiltinMacros()
	return p
}

var double = map[string]bool{
	"$$": true,
	"``": true,
	"''": true,
}

// ParseTex splits the Tokenizer's input into tokens and writes these
// tokens into the given channel.
func (p *Tokenizer) ParseTex(res chan<- *Token) error {
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return err
		}

		switch {
		case buf[0] == '\\':
			name, err := p.readMacroName()
			if err != nil {
				return err
			}

			if m := p.macros[name]; m != nil {
				tokens, err := m.ReadArgs(p, name)
				if err != nil {
					return err
				}
				for _, tok := range tokens {
					res <- tok
				}
			} else if name == "\\begin" {
				envName, err := p.readMandatoryArg()
				if err != nil {
					return err
				}
				if env := p.environments[envName]; env != nil {
					tokens, err := env.ReadArgs(p, envName)
					if err != nil {
						return err
					}
					for _, tok := range tokens {
						res <- tok
					}
				} else {
					log.Println("unknown environment", envName)
					args, err := p.readAllMacroArgs()
					if err != nil {
						return err
					}
					args = append([]*Arg{
						&Arg{
							Optional: false,
							Value:    TokenList{verbatim(envName)},
						},
					}, args...)
					res <- &Token{Type: TokenMacro, Name: name, Args: args}
				}
			} else {
				log.Println("unknown macro", name)
				args, err := p.readAllMacroArgs()
				if err != nil {
					return err
				}
				res <- &Token{Type: TokenMacro, Name: name, Args: args}
			}

		case buf[0] == '%':
			comment, err := p.readComment()
			if err != nil {
				return err
			}
			res <- &Token{Type: TokenComment, Name: comment}

		case bytes.HasPrefix(buf, []byte("\n\n")):
			err := p.skipAllWhiteSpace()
			if err != nil {
				return err
			}
			res <- &Token{Type: TokenEmptyLine}

		case isSpace(buf[0]):
			emptyLine, err := p.skipWhiteSpace()
			if err != nil {
				return err
			}
			if !emptyLine {
				res <- &Token{Type: TokenSpace}
			}

		case isLetter(buf[0]):
			word, err := p.readWord()
			if err != nil {
				return err
			}
			res <- &Token{Type: TokenWord, Name: word}

		default:
			var name string
			if len(buf) >= 2 && double[string(buf[:2])] {
				name = string(buf[:2])
				p.Skip(2)
			} else {
				name = string(buf[:1])
				p.Skip(1)
			}
			res <- &Token{Type: TokenOther, Name: name}
		}
	}
	return nil
}

func (p *Tokenizer) skipWhiteSpace() (bool, error) {
	nlSeen := 0
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return false, err
		}

		pos := 0
		for pos < len(buf) && isSpace(buf[pos]) {
			if buf[pos] == '\n' {
				nlSeen++
			}
			pos++
		}
		p.Skip(pos)
		if pos < len(buf) {
			break
		}
	}

	emptyLine := false
	if nlSeen > 1 {
		p.Prepend([]byte("\n\n"), "<end of paragraph>")
		emptyLine = true
	}
	return emptyLine, nil
}

func (p *Tokenizer) skipAllWhiteSpace() error {
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return err
		}

		pos := 0
		for pos < len(buf) && isSpace(buf[pos]) {
			pos++
		}
		p.Skip(pos)
		if pos < len(buf) {
			break
		}
	}
	return nil
}

func (p *Tokenizer) readWord() (string, error) {
	var res []byte
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		pos := 0
		for pos < len(buf) && isLetter(buf[pos]) {
			pos++
		}
		res = append(res, buf[:pos]...)
		p.Skip(pos)

		if pos < len(buf) {
			break
		}
	}
	return string(res), nil
}

func parseString(text string) TokenList {
	c := make(chan *Token, 64)
	go func() {
		p := NewTokenizer()
		p.Prepend([]byte(text), "text")
		err := p.ParseTex(c)
		if err != nil {
			// Should not happen, since the parser input is not file based.
			panic(err)
		}
		close(c)
	}()

	var res TokenList
	for tok := range c {
		res = append(res, tok)
	}
	return res
}
