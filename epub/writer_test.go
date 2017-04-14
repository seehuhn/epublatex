package epub

import (
	"io/ioutil"
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

	w, err := NewEpubWriter(out, "epubtest")
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
}
