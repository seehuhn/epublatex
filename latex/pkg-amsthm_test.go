// pkg-amsthm_test.go -
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

import (
	"testing"

	"github.com/seehuhn/epublatex/latex/tokenizer"
)

func TestNewTheorem(t *testing.T) {
	src := `\usepackage{amsthm}%
\newtheorem{theorem}{Abc}[section]
\newtheorem{lemma}[theorem]{Def}`

	conv, err := newConverter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := conv.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	toks := tokenizer.NewTokenizer()
	defer toks.Close()
	toks.Prepend([]byte(src), "test")
	err = conv.runTokenizer(toks)
	if err != nil {
		t.Fatal(err)
	}

	err = conv.Pass1()
	if err != nil {
		t.Fatal(err)
	}

	theorem := conv.Envs["theorem"]
	if theorem == nil {
		t.Fatal("theorem environment missing")
	}
	if theorem.Prefix != "Abc" {
		t.Error("wrong theorem prefix", theorem.Prefix)
	}
	lemma := conv.Envs["lemma"]
	if lemma == nil {
		t.Fatal("lemma environment missing")
	}
	if lemma.Prefix != "Def" {
		t.Error("wrong lemma prefix", theorem.Prefix)
	}
	if theorem.Counter != lemma.Counter {
		t.Error("sharing counters failed")
	}
}
