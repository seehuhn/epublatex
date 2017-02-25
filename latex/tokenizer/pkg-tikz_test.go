// pkg-tikz_test.go - unit tests for the TikZ support
// Copyright (C) 2017  Jochen Voss <voss@seehuhn.de>
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
	"testing"
)

func TestTikZ(t *testing.T) {
	tokens := parseString(`\documentclass{article}
\usepackage{tikz}
\begin{document}
Hello,
\begin{tikzpicture}
  \draw (0,0) -- (1,1);
\end{tikzpicture}
\end{document}
`)
	seen := false
	for _, tok := range tokens {
		if isMacro(tok, "%tikz%") {
			seen = true
		}
		if isMacro(tok, "\\draw") {
			t.Error("tikzpicture environment didn't capture it's contents")
		}
	}
	if !seen {
		t.Error("tikzpicture environment not detected")
	}
}
