// pkg-amsthm.go - handle the "amsthm" LaTeX package
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

func addAmsthmMacros(conv *converter, options string) {
	conv.PkgState["amsthm@style"] = "plain"
	conv.Macros["\\newtheorem"] = funcMacro(mNewtheorem)
	conv.Macros["\\theoremstyle"] = funcMacro(mTheoremstyle)
}

func mNewtheorem(args []*tokenizer.Arg, conv *converter) string {
	name := args[1].String()
	counter := args[2].String()
	if counter == "" {
		counter = name
	}
	counterName := "amsthm@" + counter

	if counter == name {
		conv.Counters[counterName] = &counterInfo{Parent: args[4].String()}
	}
	conv.Envs[name] = &environment{
		CSSClasses: []string{"amsthm-" + conv.PkgState["amsthm@style"]},
		Prefix:     args[3].String(),
		Counter:    counterName,
	}
	return ""
}

func mTheoremstyle(args []*tokenizer.Arg, conv *converter) string {
	conv.PkgState["amsthm@style"] = args[0].String()
	return ""
}

func init() {
	addPackage("amsthm", addAmsthmMacros)
}
