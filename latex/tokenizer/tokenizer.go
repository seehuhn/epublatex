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
	"io"
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

type collectState struct {
	lookingFor     isEnd
	collectingInto *Token
}

// ParseTex splits the Tokenizer's input into tokens and writes these
// tokens to the channel `res`.
func (p *Tokenizer) ParseTex(res chan<- *Token) error {
	var stack []collectState
	var lookingFor isEnd
	var collectingInto *Token

	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return err
		}

		var nextBatch TokenList

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
				nextBatch = tokens
			} else if name == "\\begin" {
				envName, err := p.readMandatoryArg()
				if err != nil {
					return err
				}
				if env := p.environments[envName]; env != nil {
					tokens, newLookingFor, err := env.ReadArgs(p, envName)
					if err != nil {
						return err
					}
					nextBatch = tokens

					if newLookingFor != nil {
						stack = append(stack,
							collectState{lookingFor, collectingInto})
					}
					lookingFor = newLookingFor
					collectingInto = nil
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
					nextBatch = TokenList{
						&Token{Type: TokenMacro, Name: name, Args: args},
					}
				}
			} else {
				log.Println("unknown macro", name)
				args, err := p.readAllMacroArgs()
				if err != nil {
					return err
				}
				nextBatch = TokenList{
					&Token{Type: TokenMacro, Name: name, Args: args},
				}
			}

		case buf[0] == '%':
			comment, err := p.readComment()
			if err != nil {
				return err
			}
			nextBatch = TokenList{&Token{Type: TokenComment, Name: comment}}

		case bytes.HasPrefix(buf, []byte("\n\n")):
			err := p.skipAllWhiteSpace()
			if err != nil {
				return err
			}
			nextBatch = TokenList{&Token{Type: TokenEmptyLine}}

		case isSpace(buf[0]):
			emptyLine, err := p.skipWhiteSpace()
			if err != nil {
				return err
			}
			if !emptyLine {
				nextBatch = TokenList{&Token{Type: TokenSpace}}
			}

		case isLetter(buf[0]):
			word, err := p.readWord()
			if err != nil {
				return err
			}
			nextBatch = TokenList{&Token{Type: TokenWord, Name: word}}

		default:
			var name string
			if len(buf) >= 2 && double[string(buf[:2])] {
				name = string(buf[:2])
				p.Skip(2)
			} else {
				name = string(buf[:1])
				p.Skip(1)
			}
			nextBatch = TokenList{&Token{Type: TokenOther, Name: name}}
		}

		for _, tok := range nextBatch {
			if lookingFor != nil {
				if collectingInto == nil {
					tok.Args = append(tok.Args, &Arg{})
					collectingInto = tok
					continue
				}
				if !lookingFor(tok) {
					k := len(collectingInto.Args) - 1
					collectingInto.Args[k].Value = append(collectingInto.Args[k].Value, tok)
					continue
				}
				tok = collectingInto

				var old collectState
				old, stack = stack[len(stack)-1], stack[:len(stack)-1]
				lookingFor = old.lookingFor
				collectingInto = old.collectingInto
			}
			res <- tok
		}
	}

	if lookingFor != nil {
		return MissingEndError(collectingInto.Name)
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

func (p *Tokenizer) readUntilChar(stopChar byte) (string, error) {
	var res []byte
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		pos := 0
		done := false
		for pos < len(buf) {
			c := buf[pos]
			pos++
			if c == stopChar {
				done = true
				break
			} else if c == '\n' {
				return "", p.MakeError("unexpected end of line")
			}
		}
		res = append(res, buf[:pos]...)
		p.Skip(pos)

		if done {
			return string(res[:len(res)-1]), nil
		}
	}
	return "", io.EOF
}

func (p *Tokenizer) readBalancedUntil(stopChar byte) (string, error) {
	var res []byte
	level := 0
	quoted := false
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		pos := 0
		done := false
		for pos < len(buf) {
			c := buf[pos]
			pos++

			if quoted {
				quoted = false
				continue
			}
			if level <= 0 && c == stopChar {
				done = true
				break
			}

			if c == '{' {
				level++
			} else if c == '}' {
				level--
			} else if c == '\\' {
				quoted = true
			}
		}
		res = append(res, buf[:pos]...)
		p.Skip(pos)

		if done {
			return string(res[:len(res)-1]), nil
		}
	}
	return "", io.EOF
}

func (p *Tokenizer) readUntilString(endMarker string) (string, error) {
	var res []byte
	endBytes := []byte(endMarker)
	done := false
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		pos := bytes.Index(buf, endBytes)
		extra := 0
		if pos >= 0 {
			done = true
			extra = len(endBytes)
		} else {
			pos = len(buf) - len(endBytes) + 1
		}
		res = append(res, buf[:pos]...)
		p.Skip(pos + extra)

		if done {
			return string(res), nil
		}
	}
	return "", io.EOF
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

func (p *Tokenizer) readNumber() (string, error) {
	var res []byte
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		pos := 0
		for pos < len(buf) &&
			(isDigit(buf[pos]) || buf[pos] == '.' || buf[pos] == '-') {
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

var isUnit = map[string]bool{
	"pt":    true,
	"pc":    true,
	"bp":    true,
	"in":    true,
	"cm":    true,
	"mm":    true,
	"dd":    true,
	"cc":    true,
	"sp":    true,
	"ex":    true,
	"em":    true,
	"fil":   true,
	"fill":  true,
	"filll": true,
}

func (p *Tokenizer) readUnit() (string, error) {
	if !p.Next() {
		return "", io.EOF
	}

	buf, err := p.Peek()
	if err != nil {
		return "", err
	}

	l := 0
	for l < len(buf) && isLetter(buf[l]) {
		l++
	}
	word := string(buf[:l])
	if !isUnit[word] {
		var next string
		if len(buf) > 13 {
			next = string(buf[:10]) + "..."
		} else {
			next = string(buf)
		}
		return "", p.MakeError("expected unit, got " + next)
	}
	p.Skip(l)

	_, err = p.skipWhiteSpace()
	if err != nil {
		return "", err
	}

	return word, nil
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
