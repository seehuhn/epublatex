// pkg-amsmath.go -
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

func addAmsmathMacros(p *Tokenizer) {
	p.macros["\\DeclareMathOperator"] = macroFunc(amsmathDMO)
	p.macros["\\eqref"] = &defMacro{Count: 1, Body: "(\ref{#1})"}

	p.environments["align"] = simpleEnv
	p.environments["align*"] = simpleEnv
	p.environments["equation*"] = simpleEnv
}

func amsmathDMO(p *Tokenizer, name string) (TokenList, error) {
	star, err := p.readOptionalStar()
	if err != nil {
		return nil, err
	}
	operatorName, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}
	value, err := p.readMandatoryArg()
	if err != nil {
		return nil, err
	}

	p.macros[operatorName] = typedMacro("")

	tok := &Token{
		Type: TokenMacro,
		Name: name,
		Args: []*Arg{
			&Arg{Optional: true, Value: star},
			&Arg{Optional: false, Value: TokenList{verbatim(operatorName)}},
			&Arg{Optional: false, Value: TokenList{verbatim(value)}},
		},
	}
	return TokenList{tok}, nil
}
