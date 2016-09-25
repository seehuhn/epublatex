// xref.go -
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

import "strconv"

type xRef struct {
	Label   string
	Chapter int
	ID      string
	Pos     int
	Type    string
	Name    string
}

func xRefNormalise(label string, used []*xRef) string {
	var chars []byte
	hyphenSeen := false
	for i := 0; i < len(label); i++ {
		c := label[i]
		if !(isLetter(c) || isDigit(c) || c == '_' || c == ':' || c == '.') {
			c = '-'
		}
		if c == '-' && hyphenSeen {
			continue
		}
		if len(chars) == 0 && !isLetter(c) {
			chars = append(chars, 'x')
		}
		chars = append(chars, c)
		hyphenSeen = c == '-'
	}
	base := string(chars)
	if base == "" {
		base = "x"
	}

	res := base
	sfx := 2
retry:
	for _, xr := range used {
		if xr.ID == res {
			res = base + strconv.Itoa(sfx)
			sfx++
			goto retry
		}
	}
	return res
}

func (conv *converter) xRefLookup(pos int) string {
	for _, label := range conv.Labels {
		if label.Pos == pos {
			return label.ID
		}
	}
	return ""
}
