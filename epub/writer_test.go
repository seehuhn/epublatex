package epub

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestWriterSimple(t *testing.T) {
	out, err := ioutil.TempFile("", "epubtest")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = out.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	settings := &Settings{
		TemplateDir: filepath.Join("..", "tmpl"),
	}
	w, err := NewEpubWriter(out, "epubtest", settings)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Flush()
	if err != nil {
		t.Fatal(err)
	}
}
