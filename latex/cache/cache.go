// cache.go - Implement the Cache object.
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
	"encoding/base64"
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

var cacheDir = flag.String("cache-dir", "",
	"cache directory for rendered images")

// Cache provides a facility to temporarily store images on disk for
// later retrival.
type Cache struct {
	cacheDir string
	entries  map[string]*entry
	start    time.Time
}

// NewCache creates a new cache, backed by subdirectory 'subdir'
// inside the cache directory.  The cache is pre-populated with any
// images found in this directory.
func NewCache(subdir string) (*Cache, error) {
	c := &Cache{
		entries: make(map[string]*entry),
		start:   time.Now(),
	}

	cacheDir := *cacheDir
	if len(cacheDir) == 0 {
		cacheDir = os.Getenv("EPUBLATEX_CACHE")
	}
	if len(cacheDir) == 0 {
		cacheDir = os.ExpandEnv(defaultCacheDir)
		cacheDir = filepath.Join(cacheDir, "de.seehuhn.ebook")
	}
	c.cacheDir = filepath.Join(cacheDir, subdir)
	err := os.MkdirAll(c.cacheDir, 0755)
	if err != nil {
		return nil, err
	}

	dir, err := os.Open(c.cacheDir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	files, err := dir.Readdir(0)
	var total int64
dirLoop:
	for _, fi := range files {
		name := fi.Name()
		if fi.IsDir() || !strings.HasSuffix(name, ".png") {
			log.Printf("cache %s: unexpected file %q", c.cacheDir, name)
			continue dirLoop
		}
		hash := name[:len(name)-4]
		e := &entry{
			Size: fi.Size(),
			Time: fi.ModTime(),
		}
		c.entries[hash] = e
		total += e.Size
	}
	log.Printf("cache %s: %s (%d objects)",
		c.cacheDir, byteSize(total), len(c.entries))

	return c, nil
}

// Close must be called when the cache is no longer needed.  Up to
// 'pruneLimit' bytes of images may be left behind in the cache
// directory; these files will be used to pre-populate future Cache
// instances.
//
// If pruneLimit >= 0, images added using the current Cache instance
// will always be retained, even if their total size exceeds
// pruneLimit.  If pruneLimit < 0, all cached data is removed.
func (c *Cache) Close(pruneLimit int64) error {
	var of oldestFirst
	var total int64
	for hash, e := range c.entries {
		of = append(of, pruneEntry{key: hash, entry: e})
		total += e.Size
	}
	sort.Sort(of)

	var err error
	var pruneCount int
	var pruneBytes int64
	for _, pe := range of {
		if total <= pruneLimit {
			break
		}
		if pruneLimit >= 0 && c.start.Before(pe.Time) {
			break
		}
		e2 := os.Remove(c.filePath(pe.key))
		if err == nil {
			err = e2
		}
		pruneCount++
		pruneBytes += pe.Size
		total -= pe.Size
	}
	if pruneCount > 0 {
		log.Printf("cache %s: removed %s (%d objects)",
			c.cacheDir, byteSize(pruneBytes), pruneCount)
	}

	if pruneLimit < 0 {
		_ = os.Remove(c.cacheDir)
	}

	c.entries = nil
	return err
}

// Has returns true, if the cache contains an image which has
// previously been stored for the given key.  The image can be
// retrieved using the .Get() method.
func (c *Cache) Has(key string) bool {
	hash := hashKey(key)
	entry, ok := c.entries[hash]
	if ok {
		entry.Time = time.Now()
	}
	return ok
}

// Put stores a new image in the cache.  The image can later be
// retrieved using the given key.  Any preexisting image using the
// same key is overwritten by subsequent calls to .Put().
func (c *Cache) Put(key string, img image.Image) (err error) {
	hash := hashKey(key)
	path := c.filePath(hash)
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		e2 := w.Close()
		if err == nil {
			err = e2
		}
	}()
	err = png.Encode(w, img)
	if err != nil {
		return err
	}

	fi, err := w.Stat()
	if err != nil {
		return err
	}
	e := &entry{
		Size: fi.Size(),
		Time: time.Now(),
	}
	c.entries[hash] = e
	return nil
}

// Get returns an image which has previously been stored in the cache
// for the given key.
func (c *Cache) Get(key string) (image.Image, error) {
	hash := hashKey(key)
	path := c.filePath(hash)
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	c.entries[hash].Time = time.Now()
	return png.Decode(in)
}

func (c *Cache) filePath(hash string) string {
	return filepath.Join(c.cacheDir, hash+".png")
}

func hashKey(key string) string {
	h := sha3.NewShake128()
	h.Write([]byte(key))
	buf := make([]byte, 15)
	h.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

type entry struct {
	Size int64
	Time time.Time
}

type pruneEntry struct {
	key string
	*entry
}

type oldestFirst []pruneEntry

func (of oldestFirst) Len() int { return len(of) }
func (of oldestFirst) Less(i, j int) bool {
	return of[i].Time.Before(of[j].Time)
}
func (of oldestFirst) Swap(i, j int) { of[i], of[j] = of[j], of[i] }
