// section.go - handle section numbers
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

import (
	"strconv"
	"strings"
)

// SecNo represents a section number in a document, e.g. `SecNo{1, 2}`
// represents section 1.2 and `SecNo{1}` represents chapter 1.
type SecNo []int

// String formats a section number as a string, in the form "1.2.3".
func (s SecNo) String() string {
	var parts []string
	for _, k := range s {
		parts = append(parts, strconv.Itoa(k))
	}
	return strings.Join(parts, ".")
}

// Inc increases the section number at a given level by 1.  Level
// should be 1 to increase the chapter number, 2 to increase the
// section number etc.
func (s *SecNo) Inc(level int) {
	if len(*s) > level {
		*s = (*s)[:level]
	} else {
		for len(*s) < level {
			*s = append(*s, 0)
		}
	}
	(*s)[level-1]++
}
