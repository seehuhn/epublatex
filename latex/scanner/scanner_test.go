// scanner_test.go -
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

package scanner

import (
	"errors"
	"testing"
)

func TestScannerSimple(t *testing.T) {
	scan := &Scanner{}
	target := "testing"
	scan.Prepend([]byte(target[4:]), "end")
	scan.Prepend([]byte(target[:4]), "beginning")

	for len(target) > 0 {
		hasData := scan.Next()
		if !hasData {
			t.Fatal("unexpected end of data")
		}
		buf, err := scan.Peek()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if string(buf) != target {
			t.Fatalf("expected %q, got %q", target, string(buf))
		}
		scan.Skip(1)
		target = target[1:]
	}

	hasData := scan.Next()
	if hasData {
		t.Fatal("unexpected data")
	}
}

func TestScannerError(t *testing.T) {
	scan := &Scanner{}
	scan.Prepend([]byte("\nline after include\nend\n"), "level1")
	scan.Prepend([]byte("line 1\nline 2\nlin"), "level2")
	scan.sources[1].err = errors.New("something bad happened")
	scan.Prepend([]byte("some\nincluded\nstuff\n"), "level3")

	for scan.Next() {
		buf, err := scan.Peek()
		if err != nil {
			e2, ok := err.(*ParseError)
			if !ok {
				t.Fatalf("wrong error %q", err)
			}
			if e2.stack[0].Name != "level2" || e2.stack[0].Line != 3 {
				t.Fatalf("wrong error location in %q", err)
			}
			return
		}
		scan.Skip(len(buf))
	}
	t.Fatal("error not reported")
}
