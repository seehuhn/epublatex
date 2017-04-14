// section_test.go - unit tests for section.go
//
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

package epub

import "testing"

func TestSecNoInc(t *testing.T) {
	s := SecNo{}
	s.Inc(1) // 1
	if len(s) != 1 || s[0] != 1 {
		t.Fatal("Inc(1) failed,", s)
	}
	s.Inc(1) // 2
	if len(s) != 1 || s[0] != 2 {
		t.Fatal("Inc(1) failed,", s)
	}
	s.Inc(3) // 2.0.1
	if len(s) != 3 || s[0] != 2 || s[1] != 0 || s[2] != 1 {
		t.Fatal("Inc(3) failed,", s)
	}
	s.Inc(2) // 2.1
	if len(s) != 2 || s[0] != 2 || s[1] != 1 {
		t.Fatal("Inc(2) failed,", s)
	}
}
