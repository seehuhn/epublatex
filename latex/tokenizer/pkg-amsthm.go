// pkg-amsthm.go -
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

func addAmsthmMacros(p *Tokenizer) {
	p.macros["\\newtheorem"] = macroFunc(parseNewtheorem)
	p.macros["\\theoremstyle"] = typedMacro("V")
}

func parseNewtheorem(p *Tokenizer, name string) (TokenList, error) {
	star, err := p.readOptionalStar()
	if err != nil {
		return nil, err
	}
	arg1, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}
	arg2, err := p.readOptionalArg()
	if err != nil {
		return nil, err
	}
	arg3, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}
	arg4, err := p.readOptionalArg()
	if err != nil {
		return nil, err
	}

	p.environments[arg1] = typedEnv("O")

	tok := &Token{
		Type: TokenMacro,
		Name: name,
		Args: []*Arg{
			&Arg{Optional: true, Value: star},
			&Arg{Optional: false, Value: TokenList{verbatim(arg1)}},
			&Arg{Optional: true, Value: TokenList{verbatim(arg2)}},
			&Arg{Optional: false, Value: TokenList{verbatim(arg3)}},
			&Arg{Optional: true, Value: TokenList{verbatim(arg4)}},
		},
	}
	return TokenList{tok}, nil
}
