// env.go -
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

import "github.com/seehuhn/epublatex/latex/tokenizer"

type environment struct {
	CSSClasses []string
	Prefix     string
	Counter    string

	RenderMath string
}

type isEnd func(token *tokenizer.Token) bool

func (conv *converter) IsMathStart(token *tokenizer.Token) (string, isEnd) {
	if token.Type == tokenizer.TokenOther && token.Name == "$" {
		endFn := func(token *tokenizer.Token) bool {
			return token.Type == tokenizer.TokenOther && token.Name == "$"
		}
		return "$", endFn
	}
	if token.Type != tokenizer.TokenMacro || token.Name != "\\begin" {
		return "", nil
	}

	envName := token.Args[0].String()
	env := conv.Envs[envName]
	if env == nil || env.RenderMath == "" {
		return "", nil
	}

	endFn := func(token *tokenizer.Token) bool {
		if token.Type != tokenizer.TokenMacro || token.Name != "\\end" {
			return false
		}
		return token.Args[0].String() == envName
	}
	return env.RenderMath, endFn
}
