// cache_test.go - unit tests for cache.go
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

package cache

import (
	"crypto/rand"
	"image"
	"os"
	"testing"
)

func TestCache(t *testing.T) {
	c, err := NewCache("test")
	if err != nil {
		t.Fatal(err)
	}

	rect := image.Rect(0, 0, 10, 10)
	img := image.NewRGBA(rect)
	rand.Read(img.Pix)

	err = c.Put("A", img)
	if err != nil {
		t.Error(err)
	}

	if !c.Has("A") {
		t.Error("key A not found")
	}
	if c.Has("B") {
		t.Error("non-existent key B found")
	}

	i2, err := c.Get("A")
	if !img.Bounds().Eq(i2.Bounds()) {
		t.Error("key A yielded wrong image size")
	}
	i2, err = c.Get("B")
	if !os.IsNotExist(err) {
		t.Error("requesting non-existent image B returned wrong error", err)
	}

	err = c.Close(-1)
	if err != nil {
		t.Fatal(err)
	}
}
