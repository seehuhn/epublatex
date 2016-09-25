// tokenizer_test.go -
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
	"encoding/gob"
	"os"
	"testing"
)

func TestParseTex(t *testing.T) {
	p := NewTokenizer()
	p.Include("/Users/voss/Sync/tree/tree.tex")

	c := make(chan *Token)
	go func() {
		err := p.ParseTex(c)
		close(c)
		if err != nil {
			t.Fatal(err)
		}
	}()

	out, err := os.Create("test.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	enc := gob.NewEncoder(out)
	for tok := range c {
		err := enc.Encode(tok)
		if err != nil {
			t.Fatal(err)
		}
	}
}
