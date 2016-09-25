// environments.go -
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

type environment interface {
	ReadArgs(p *Tokenizer, name string) (TokenList, error)
}

type envFunc func(p *Tokenizer, name string) (TokenList, error)

type simpleEnvClass struct{}

func (env simpleEnvClass) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	tok := &Token{
		Type: TokenMacro,
		Name: "\\begin",
		Args: []*Arg{
			&Arg{
				Optional: false,
				Value:    TokenList{verbatim(name)},
			},
		},
	}
	return TokenList{tok}, nil
}

var simpleEnv = simpleEnvClass{}

type typedEnv string

func (env typedEnv) ReadArgs(p *Tokenizer, name string) (TokenList, error) {
	args := []*Arg{
		&Arg{
			Optional: false,
			Value:    TokenList{verbatim(name)},
		},
	}
	for _, argType := range env {
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
	return TokenList{&Token{Type: TokenMacro, Name: "\\begin", Args: args}}, nil
}
