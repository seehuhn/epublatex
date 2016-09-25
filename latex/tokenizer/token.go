// token.go -
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
	"strconv"
	"strings"
)

// TokenType is used to enumerate different types of token
type TokenType int

// The different token types used by this package.
const (
	TokenMacro TokenType = iota
	TokenEmptyLine
	TokenComment
	TokenSpace
	TokenWord
	TokenOther
	TokenVerbatim
)

// Token contains information about a single syntactic unit in the TeX
// source.
type Token struct {
	// Type describes which kind of token this is.
	Type TokenType

	// For TokenMacro, this is the name of the macro, including the
	// leading backslash.  For most other token types, this is the
	// textual content of the token.
	Name string

	// For tokens of type TokenMacro, this field specifies the values
	// of the macro arguments.  Unused for all other token types.
	Args []*Arg
}

// Arg specifies a single macro argument.
type Arg struct {
	Optional bool
	Value    TokenList
}

func (arg *Arg) String() string {
	return arg.Value.FormatText()
}

func verbatim(s string) *Token {
	return &Token{
		Type: TokenVerbatim,
		Name: s,
	}
}

// TokenList describes tokenized data in the argument of a macro call.
type TokenList []*Token

func (toks TokenList) FormatText() string {
	var res []string
	mayNeedSpace := false
	for _, tok := range toks {
		switch tok.Type {
		case TokenMacro:
			res = append(res, tok.Name)
			for _, arg := range tok.Args {
				text := arg.Value.FormatText()
				if arg.Optional {
					if text != "" {
						text = "[" + text + "]"
					}
				} else {
					text = "{" + text + "}"
				}
				res = append(res, text)
			}
		case TokenComment:
			// pass
		case TokenSpace:
			res = append(res, " ")
		case TokenWord:
			if mayNeedSpace {
				res = append(res, " "+tok.Name)
			} else {
				res = append(res, tok.Name)
			}
		case TokenOther, TokenVerbatim:
			res = append(res, tok.Name)
		default:
			panic("invalid token type " + strconv.Itoa(int(tok.Type)))
		}
		mayNeedSpace = tok.Type == TokenMacro && len(tok.Args) == 0
	}
	return strings.Join(res, "")
}

func (toks TokenList) FormatMaths() string {
	var res []string
	mayNeedSpace := false
	for _, tok := range toks {
		switch tok.Type {
		case TokenMacro:
			res = append(res, tok.Name)
			for _, arg := range tok.Args {
				if tok.Name == "\\mbox" {
					res = append(res, "{"+arg.Value.FormatText()+"}")
				} else if arg.Optional {
					val := arg.Value.FormatMaths()
					if val != "" {
						res = append(res, "["+val+"]")
					}
				} else {
					res = append(res, "{"+arg.Value.FormatMaths()+"}")
				}
			}
		case TokenComment, TokenSpace:
			// pass
		case TokenWord:
			if mayNeedSpace {
				res = append(res, " "+tok.Name)
			} else {
				res = append(res, tok.Name)
			}
		case TokenOther, TokenVerbatim:
			res = append(res, tok.Name)
		default:
			panic("invalid token type " + strconv.Itoa(int(tok.Type)))
		}
		if tok.Type != TokenSpace {
			mayNeedSpace = tok.Type == TokenMacro && len(tok.Args) == 0
		}
	}
	return strings.Join(res, "")
}
