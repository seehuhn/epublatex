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

package latex

func addAmsmathMacros(conv *converter, options string) {
	// TODO(voss): copy this into the TeX file
	conv.Macros["\\DeclareMathOperator"] = mIgnore

	conv.Envs["equation*"] = &environment{
		RenderMath: "equation*",
	}
	conv.Envs["align*"] = &environment{
		RenderMath: "align*",
	}
}

func init() {
	addPackage("amsmath", addAmsmathMacros)
}
