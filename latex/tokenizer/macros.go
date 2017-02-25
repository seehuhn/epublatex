// macros.go -
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
	"strconv"
	"strings"

	"github.com/seehuhn/epublatex/latex/scanner"
)

type pkgInitFunc func(p *Tokenizer)

var pkgInit map[string]pkgInitFunc

func addPackage(name string, init pkgInitFunc) {
	if pkgInit == nil {
		pkgInit = make(map[string]pkgInitFunc)
	}
	pkgInit[name] = init
}

type macro interface {
	ReadArgs(p *Tokenizer, name string) (TokenList, error)
}

type macroFunc func(p *Tokenizer, name string) (TokenList, error)

func (mf macroFunc) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	return mf(p, name)
}

func (p *Tokenizer) addBuiltinMacros() {
	// builtin EPUB support
	p.macros["\\epubauthor"] = typedMacro("A")
	p.macros["\\epubcover"] = typedMacro("A")
	p.macros["\\epubmaketitle"] = typedMacro("")
	p.macros["\\epubsection"] = typedMacro("OA")
	p.macros["\\epubsubsection"] = typedMacro("OA")
	p.macros["\\epubtitle"] = typedMacro("A")

	// TeX/LaTeX macros
	p.macros["\\ "] = &defMacro{Count: 0, Body: " "}
	p.macros["\\,"] = typedMacro("")
	p.macros["\\\\"] = typedMacro("O")
	p.macros["\\alpha"] = typedMacro("")
	p.macros["\\approx"] = typedMacro("")
	p.macros["\\beta"] = typedMacro("")
	p.macros["\\bigl"] = typedMacro("")
	p.macros["\\bigm"] = typedMacro("")
	p.macros["\\bigr"] = typedMacro("")
	p.macros["\\chi"] = typedMacro("")
	p.macros["\\colon"] = typedMacro("")
	p.macros["\\def"] = macroFunc(parseDef)
	p.macros["\\delta"] = typedMacro("")
	p.macros["\\documentclass"] = macroFunc(parseDocumentclass)
	p.macros["\\end"] = typedMacro("V")
	p.macros["\\epsilon"] = typedMacro("")
	p.macros["\\eta"] = typedMacro("")
	p.macros["\\frac"] = typedMacro("AA")
	p.macros["\\gamma"] = typedMacro("")
	p.macros["\\hskip"] = macroFunc(parseHskip)
	p.macros["\\in"] = typedMacro("")
	p.macros["\\infty"] = typedMacro("")
	p.macros["\\int"] = typedMacro("")
	p.macros["\\iota"] = typedMacro("")
	p.macros["\\kappa"] = typedMacro("")
	p.macros["\\label"] = typedMacro("V")
	p.macros["\\lambda"] = typedMacro("")
	p.macros["\\ldots"] = typedMacro("")
	p.macros["\\mathcal"] = typedMacro("")
	p.macros["\\mbox"] = typedMacro("A")
	p.macros["\\mu"] = typedMacro("")
	p.macros["\\neq"] = typedMacro("")
	p.macros["\\nu"] = typedMacro("")
	p.macros["\\omega"] = typedMacro("")
	p.macros["\\phi"] = typedMacro("")
	p.macros["\\pi"] = typedMacro("")
	p.macros["\\pi"] = typedMacro("")
	p.macros["\\psi"] = typedMacro("")
	p.macros["\\ref"] = typedMacro("V")
	p.macros["\\rho"] = typedMacro("")
	p.macros["\\sigma"] = typedMacro("")
	p.macros["\\sum"] = typedMacro("")
	p.macros["\\tau"] = typedMacro("")
	p.macros["\\textit"] = typedMacro("A")
	p.macros["\\theta"] = typedMacro("")
	p.macros["\\times"] = typedMacro("")
	p.macros["\\to"] = typedMacro("")
	p.macros["\\usepackage"] = macroFunc(parseUsepackage)
	p.macros["\\varepsilon"] = typedMacro("")
	p.macros["\\varphi"] = typedMacro("")
	p.macros["\\verb"] = macroFunc(parseVerb)
	p.macros["\\xi"] = typedMacro("")
	p.macros["\\zeta"] = typedMacro("")
	p.macros["\\{"] = typedMacro("")
	p.macros["\\}"] = typedMacro("")

	p.environments["document"] = simpleEnv
	p.environments["equation"] = simpleEnv
	p.environments["verbatim"] = verbatimEnv("%verbatim%")
}

func parseDocumentclass(p *Tokenizer, name string) (TokenList, error) {
	options, err := p.readOptionalArg()
	if err != nil {
		return nil, err
	}
	class, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}

	switch class {
	case "article":
		p.macros["\\author"] = letMacro("\\epubauthor")
		p.macros["\\maketitle"] = letMacro("\\epubmaketitle")
		p.macros["\\section"] = letMacro("\\epubsection")
		p.macros["\\subsection"] = letMacro("\\epubsubsection")
		p.macros["\\title"] = letMacro("\\epubtitle")
	default:
		log.Println("unknown document class", class)
	}

	tok := &Token{
		Type: TokenMacro,
		Name: name,
		Args: []*Arg{
			&Arg{
				Optional: true,
				Value:    TokenList{verbatim(options)},
			},
			&Arg{
				Optional: false,
				Value:    TokenList{verbatim(class)},
			},
		},
	}
	return TokenList{tok}, nil
}

func parseUsepackage(p *Tokenizer, name string) (TokenList, error) {
	var res TokenList

	options, err := p.readOptionalArg()
	if err != nil {
		return nil, err
	}
	packages, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}

	for _, pkg := range strings.Split(packages, ",") {
		pkg = strings.TrimSpace(pkg)

		load := pkgInit[pkg]
		if load != nil {
			load(p)
		} else {
			log.Printf("unknown usepackage %q", pkg)
		}

		tok := &Token{
			Type: TokenMacro,
			Name: name,
			Args: []*Arg{
				&Arg{
					Optional: true,
					Value:    TokenList{verbatim(options)},
				},
				&Arg{
					Optional: false,
					Value:    TokenList{verbatim(pkg)},
				},
			},
		}
		res = append(res, tok)
	}
	return res, nil
}

func parseDef(p *Tokenizer, _ string) (TokenList, error) {
	defName, err := p.readMacroName()
	if err != nil {
		return nil, err
	}

	count := 0
	idx := 1
	for p.Next() {
		iStr := "#" + strconv.Itoa(idx)
		idx++

		buf, err := p.Peek()
		if err != nil {
			return nil, err
		}
		if bytes.HasPrefix(buf, []byte(iStr)) {
			count++
			p.Skip(len(iStr))
		} else {
			break
		}
	}

	body, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}

	// Macro names starting with "\epub" cannot be redefined.
	if !strings.HasPrefix(defName, "\\epub") {
		p.macros[defName] = &defMacro{
			Count: count,
			Body:  body,
		}
	}
	return nil, nil
}

func parseHskip(p *Tokenizer, name string) (TokenList, error) {
	amount, err := p.readNumber()
	if err != nil {
		return nil, err
	}
	_, err = p.skipWhiteSpace()
	if err != nil {
		return nil, err
	}
	unit, err := p.readUnit()
	if err != nil {
		return nil, err
	}
	tok := &Token{
		Type: TokenMacro,
		Name: name,
		Args: []*Arg{
			&Arg{
				Value: TokenList{verbatim(amount + unit)},
			},
		},
	}
	return TokenList{tok}, nil
}

func parseVerb(p *Tokenizer, name string) (TokenList, error) {
	if !p.Next() {
		return nil, io.EOF
	}
	buf, err := p.Peek()
	if err != nil {
		return nil, err
	}
	sep := buf[0]
	p.Skip(1)
	body, err := p.readUntilChar(sep)
	if err != nil {
		return nil, err
	}

	tok := &Token{
		Type: TokenMacro,
		Name: name,
		Args: []*Arg{
			&Arg{
				Optional: false,
				Value:    TokenList{verbatim(body)},
			},
		},
	}
	return TokenList{tok}, nil
}

type letMacro string

func (m letMacro) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	p.Prepend([]byte(m), name+" -> "+string(m))
	return nil, nil
}

type defMacro struct {
	Count int
	Body  string
}

func (dm *defMacro) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	args := make([]string, dm.Count)
	for i := range args {
		arg, err := p.readMandatoryArg()
		if err != nil {
			return nil, err
		}
		args[i] = arg
	}
	out := substituteMacroArgs(dm.Body, args)
	p.Prepend([]byte(out), name+" macro body")
	return nil, nil
}

func substituteMacroArgs(body string, args []string) string {
	var parts []string

	partStart := 0
	numStart := -1
	hashSeen := false
	for pos := 0; pos < len(body); pos++ {
		c := body[pos]

		if numStart >= 0 {
			if isDigit(c) {
				continue
			}

			num, err := strconv.Atoi(body[numStart:pos])
			if err == nil && num > 0 && num <= len(args) {
				parts = append(parts, args[num-1])
			}
			partStart = pos
			numStart = -1
		}

		switch {
		case hashSeen && isDigit(c):
			numStart = pos
			hashSeen = false
		case c == '#' && !hashSeen:
			parts = append(parts, body[partStart:pos])
			partStart = pos + 1
			hashSeen = true
		default:
			hashSeen = false
		}
	}
	parts = append(parts, body[partStart:])
	return strings.Join(parts, "")
}

type typedMacro string

func (tm typedMacro) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	var args []*Arg
	for _, argType := range tm {
		switch argType {
		case 'A':
			arg, err := p.readMandatoryArg()
			if err != nil {
				return nil, err
			}
			args = append(args, &Arg{Optional: false, Value: parseString(arg)})
		case 'O':
			arg, err := p.readOptionalArg()
			if err != nil {
				return nil, err
			}
			args = append(args, &Arg{Optional: true, Value: parseString(arg)})
		case 'V':
			arg, err := p.readMandatoryArg()
			if err != nil {
				return nil, err
			}
			args = append(args, &Arg{
				Optional: false,
				Value:    TokenList{verbatim(arg)},
			})
		}
	}
	return TokenList{&Token{Type: TokenMacro, Name: name, Args: args}}, nil
}

func (p *Tokenizer) readMacroName() (string, error) {
	if !p.Next() {
		return "", io.EOF
	}
	buf, err := p.Peek()
	if err != nil {
		return "", err
	}
	if buf[0] != '\\' {
		defer p.Skip(1)
		return string(buf[:1]), nil
	}
	if len(buf) < 2 {
		return "", io.EOF
	}
	if !isLetter(buf[1]) {
		defer p.Skip(2)
		return string(buf[:2]), nil
	}

	var i int
	for i = 1; i < len(buf); i++ {
		if !isLetter(buf[i]) {
			break
		}
	}
	if i >= scanner.PeekWindowSize {
		return "", p.MakeError("macro name too long")
	}
	name := string(buf[:i])
	p.Skip(i)

	_, err = p.skipWhiteSpace()
	return string(name), err
}

func (p *Tokenizer) readMandatoryArg() (string, error) {
	_, err := p.skipWhiteSpace()
	if err != nil {
		return "", err
	}

	if !p.Next() {
		return "", io.EOF
	}
	buf, err := p.Peek()
	if err != nil {
		return "", err
	}
	c := buf[0]
	p.Skip(1)
	if c != '{' {
		return string(c), nil
	}

	return p.readBalancedUntil('}')
}

func (p *Tokenizer) readOptionalArg() (string, error) {
	if !p.Next() {
		return "", nil
	}
	buf, err := p.Peek()
	if err != nil {
		return "", err
	}
	space := isSpace(buf[0])
	if space {
		_, err = p.skipWhiteSpace()
		if err != nil {
			return "", err
		}
	}

	if !p.Next() {
		return "", nil
	}
	buf, err = p.Peek()
	if err != nil {
		return "", err
	}
	if buf[0] != '[' {
		if space {
			p.Prepend([]byte{' '}, "space after macro")
		}
		return "", nil
	}

	p.Skip(1)
	return p.readBalancedUntil(']')
}

func (p *Tokenizer) readOptionalStar() (TokenList, error) {
	if !p.Next() {
		return nil, io.EOF
	}
	buf, err := p.Peek()
	if err != nil {
		return nil, err
	}
	var star TokenList
	if buf[0] == '*' {
		star = TokenList{&Token{Type: TokenOther, Name: "*"}}
		p.Skip(1)
	}
	return star, nil
}

func (p *Tokenizer) readAllMacroArgs() ([]*Arg, error) {
	var args []*Arg
loop:
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return nil, err
		}

		switch buf[0] {
		case '{':
			p.Skip(1)
			arg, err := p.readBalancedUntil('}')
			if err != nil {
				return nil, err
			}
			args = append(args, &Arg{Optional: false, Value: parseString(arg)})
		case '[':
			p.Skip(1)
			arg, err := p.readBalancedUntil(']')
			if err != nil {
				return nil, err
			}
			args = append(args, &Arg{Optional: true, Value: parseString(arg)})
		case '%':
			_, err := p.readComment()
			if err != nil {
				return nil, err
			}
		default:
			break loop
		}
	}

	return args, nil
}

func isMacro(tok *Token, name string, args ...string) bool {
	if tok.Type != TokenMacro {
		return false
	}
	if tok.Name != name {
		return false
	}
	if len(tok.Args) < len(args) {
		return false
	}
	for i, arg := range args {
		if tok.Args[i].String() != arg {
			return false
		}
	}
	return true
}
