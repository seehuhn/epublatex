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
	"fmt"
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

func (tok *Token) String() string {
	var tokType string
	switch tok.Type {
	case TokenMacro:
		tokType = "Macro"
	case TokenEmptyLine:
		tokType = "EmptyLine"
	case TokenComment:
		tokType = "Comment"
	case TokenSpace:
		tokType = "Space"
	case TokenWord:
		tokType = "Word"
	case TokenOther:
		tokType = "Other"
	case TokenVerbatim:
		tokType = "Verbatim"
	default:
		tokType = fmt.Sprintf("type%d", tok.Type)
	}
	if len(tok.Args) > 0 {
		return fmt.Sprintf("%s('%s' %q)", tokType, tok.Name, tok.Args)
	}
	return fmt.Sprintf("%s('%s')", tokType, tok.Name)
}

func verbatim(s string) *Token {
	return &Token{
		Type: TokenVerbatim,
		Name: s,
	}
}

// TokenList describes tokenized data in the argument of a macro call.
type TokenList []*Token

// FormatText converts a TokenList to a string.
func (toks TokenList) FormatText() string {
	var res []string
	mayNeedSpace := false
	for _, tok := range toks {
		switch tok.Type {
		case TokenMacro:
			res = append(res, tok.Name)
			for _, arg := range tok.Args {
				text := arg.Value.FormatText()
				if tok.Name == "\\hskip" {
					// pass
				} else if arg.Optional {
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
		if tok.Type != TokenSpace {
			mayNeedSpace = tok.Type == TokenMacro && len(tok.Args) == 0 ||
				tok.Type == TokenMacro && tok.Name == "\\hskip"
		}
	}
	return strings.Join(res, "")
}

// FormatMaths converts a TokenList to a string, assuming maths mode.
// Redundant spaces are omitted by this method.
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
				} else if tok.Name == "\\hskip" {
					res = append(res, arg.Value.FormatText())
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
			mayNeedSpace = tok.Type == TokenMacro && len(tok.Args) == 0 ||
				tok.Type == TokenMacro && tok.Name == "\\hskip"
		}
	}
	return strings.Join(res, "")
}

// Arg specifies a single macro argument.
type Arg struct {
	Optional bool
	Value    TokenList
}

func (arg *Arg) String() string {
	return arg.Value.FormatText()
}
