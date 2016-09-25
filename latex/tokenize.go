// tokenize.go -
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
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/seehuhn/ebook/latex/tokenizer"
)

func (conv *converter) Tokenize(inputFileName string) error {
	toks := tokenizer.NewTokenizer()
	defer toks.Close()
	toks.Include(inputFileName)
	return conv.runTokenizer(toks)
}

func (conv *converter) runTokenizer(toks *tokenizer.Tokenizer) (err error) {
	conv.TokenFileName = filepath.Join(conv.WorkDir, "tokens.dat")
	out, err := os.Create(conv.TokenFileName)
	if err != nil {
		return err
	}
	defer out.Close()

	c := make(chan *tokenizer.Token)
	errChan := make(chan error, 1)
	go func() {
		e2 := toks.ParseTex(c)
		close(c)
		errChan <- e2
	}()

	enc := gob.NewEncoder(out)
	for tok := range c {
		e2 := enc.Encode(tok)
		if err == nil {
			err = e2
		}
	}
	e2 := <-errChan
	if err == nil {
		err = e2
	}

	conv.SourceDir = toks.BaseDir

	return
}
