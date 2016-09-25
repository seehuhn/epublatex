// comment_test.go -
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
	"bytes"
	"testing"
)

func TestReadComment(t *testing.T) {
	p := NewTokenizer()
	p.Prepend([]byte("% line 1\n% line 2 \t \n\t % line 3\n   xxx"), "test")
	comment, err := p.readComment()
	if err != nil {
		t.Fatal(err)
	}
	expected := " line 1\n line 2\n line 3"
	if comment != expected {
		t.Errorf("comment failed: got %q, expected %q", comment, expected)
	}

	if !p.Next() {
		t.Fatal("unexpected EOF")
	}
	buf, err := p.Peek()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf, []byte("xxx")) {
		t.Error("wrong prefix", string(buf))
	}
}
