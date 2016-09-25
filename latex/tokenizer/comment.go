// comment.go -
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
	"strings"
	"unicode"
)

func (p *Tokenizer) readComment() (string, error) {
	var lines []string
	var parts []string

	state := 1
loop:
	for p.Next() {
		buf, err := p.Peek()
		if err != nil {
			return "", err
		}

		switch state {
		case 1: // look for '%'
			if buf[0] != '%' {
				break loop
			}
			state = 2
			fallthrough
		case 2: // look for end of line
			pos := 0
			for pos < len(buf) && buf[pos] != '\n' {
				pos++
			}
			parts = append(parts, string(buf[:pos]))
			if pos < len(buf) {
				line := strings.Join(parts, "")
				parts = nil
				line = strings.TrimRightFunc(line[1:], unicode.IsSpace)
				lines = append(lines, line)
				state = 3
			}
			p.Skip(pos)
		case 3: // skip white space
			pos := 0
			for pos < len(buf) && isSpace(buf[pos]) {
				pos++
			}
			if pos < len(buf) {
				state = 1
			}
			p.Skip(pos)
		}
	}
	return strings.Join(lines, "\n"), nil
}
