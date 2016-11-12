// scanner.go -
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PeekWindowSize gives the minimum size of the lookahead buffer.
// Unless the end of input is reached, at least this many bytes
// are visible in the buffer returned by the .Peek() method.
const PeekWindowSize = 128

const peekBufferSize = 1024

// Scanner implements methods to recursivly walk through a set of
// input files and buffers.
type Scanner struct {
	// BaseDir is the base directory for include files.  Filenames
	// passed to the .Include() method are interpreted as being
	// relative to this directory.
	BaseDir string

	sources []*source
	peekBuf []byte
	ready   bool
}

// Close closes all input files and discards all buffers used by the
// scanner.
func (scan *Scanner) Close() (err error) {
	for _, source := range scan.sources {
		e2 := source.Fd.Close()
		if err == nil {
			err = e2
		}
	}
	scan.sources = nil
	return
}

// Prepend adds the given buffer to the list of input sources.  The
// buffer contents are read next, followed by all previous inputs.
// The argument `name` is used to identify the buffer in error
// messages and should be a short, human-readable string.
func (scan *Scanner) Prepend(data []byte, name string) {
	src := &source{
		Name:   name,
		Buffer: data,
	}
	scan.sources = append(scan.sources, src)
}

// Include adds the contents of the given file to the list of input
// sources.  The file contents are read next, followed by all
// remaining, previously registered inputs.
func (scan *Scanner) Include(fileName string) error {
	if scan.BaseDir != "" {
		fileName = filepath.Join(scan.BaseDir, fileName)
	}

	fd, err := os.Open(fileName)
	if err != nil {
		return err
	}

	src := &source{
		Name: filepath.Base(fileName),
		Fd:   fd,
	}
	scan.sources = append(scan.sources, src)

	if scan.BaseDir == "" {
		tmp, err := filepath.Abs(fileName)
		if err != nil {
			return err
		}
		scan.BaseDir = filepath.Dir(tmp)
	}

	return nil
}

// Next checks whether more input is available.  This method must be
// called before every call to the .Peek() method.
func (scan *Scanner) Next() bool {
	var peekBuf []byte
	for idx := len(scan.sources) - 1; idx >= 0; idx-- {
		if len(peekBuf) >= PeekWindowSize {
			break
		}

		src := scan.sources[idx]
		if len(peekBuf)+len(src.Buffer) < PeekWindowSize &&
			src.Fd != nil &&
			src.err == nil {
			buf := make([]byte, peekBufferSize)
			n, err := src.Fd.Read(buf)
			src.Buffer = append(src.Buffer, buf[:n]...)
			if src.err == nil && err != io.EOF {
				src.err = err
			}
			if err != nil {
				err = src.Fd.Close()
				if src.err == nil {
					src.err = err
				}
				src.Fd = nil
			}
		}
		peekBuf = append(peekBuf, src.Buffer...)
		if src.err != nil {
			break
		}
	}
	scan.peekBuf = peekBuf

	n := len(scan.sources)
	for n > 0 &&
		len(scan.sources[n-1].Buffer) == 0 &&
		scan.sources[n-1].err == nil {
		n--
	}
	scan.sources = scan.sources[:n]
	scan.ready = true

	return len(peekBuf) > 0 || len(scan.sources) > 0
}

// Peek returns a buffer showing the first input bytes after the
// current input position.  Unless the end of file is reached, this
// buffer is at least PeekWindowSize bytes long.  The current input
// position is not changed by calls to .Peek().
//
// The contents of the returned buffer are only valid until the next
// call to the .Skip() method.  The .Next() method must be called to
// populate the look-ahead buffer before every call to .Peek().
func (scan *Scanner) Peek() ([]byte, error) {
	if !scan.ready {
		panic("parser not ready, missing call to .Next()")
	}
	if len(scan.peekBuf) > 0 {
		return scan.peekBuf, nil
	}
	idx := len(scan.sources) - 1
	return nil, scan.MakeError(scan.sources[idx].err.Error())
}

// Skip advances the current position in the scanner inputs by n
// bytes.
func (scan *Scanner) Skip(n int) {
	if n < 0 {
		panic("invalid skip amount")
	}
	scan.ready = false
	idx := len(scan.sources) - 1
	for n > 0 {
		src := scan.sources[idx]
		k := len(src.Buffer)
		if k > n {
			k = n
		}
		src.Skip(k)
		n -= k
		scan.peekBuf = scan.peekBuf[k:]
		idx--
	}
}

type source struct {
	Name   string
	Fd     io.ReadCloser
	Buffer []byte
	Line   int
	err    error
}

func (src *source) Skip(n int) {
	for _, c := range src.Buffer[:n] {
		if c == '\n' {
			src.Line++
		}
	}
	src.Buffer = src.Buffer[n:]
}

// MakeError returns an error object which includes the given message
// together with human-readable information about the current input
// position.
func (scan *Scanner) MakeError(message string) *ParseError {
	err := &ParseError{
		Message: message,
	}
	for idx := len(scan.sources) - 1; idx >= 0; idx-- {
		src := scan.sources[idx]
		var context string
		if len(src.Buffer) > 20 {
			context = string(src.Buffer[:17]) + "..."
		} else {
			context = string(src.Buffer)
		}
		err.stack = append(err.stack, stackFrame{
			Name:    src.Name,
			Line:    src.Line + 1,
			Context: context,
		})
	}
	return err
}

type stackFrame struct {
	Name    string
	Line    int
	Context string
}

type ParseError struct {
	Message string
	stack   []stackFrame
}

func (err *ParseError) Error() string {
	res := []string{err.Message}
	for i, frame := range err.stack {
		if i > 0 {
			res = append(res, ", included from")
		}
		res = append(res, "\n    ",
			frame.Name, ", line ", strconv.Itoa(frame.Line))
		if frame.Context != "" {
			res = append(res, fmt.Sprintf(", before %q", frame.Context))
		}
	}
	return strings.Join(res, "")
}
