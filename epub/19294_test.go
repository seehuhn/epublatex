package epub

import (
	"bytes"
	"html/template"
	"testing"
)

func TestIssue19294(t *testing.T) {
	// The empty block in "xhtml" should be replaced during execution
	// by the contents of "stylesheet", but if the internal map associating
	// names with templates is built in the wrong order, the empty block
	// looks non-empty and this doesn't happen.
	var inlined = map[string]string{
		"stylesheet": `{{define "stylesheet"}}stylesheet{{end}}`,
		"xhtml":      `{{block "stylesheet" .}}{{end}}`,
	}
	all := []string{"stylesheet", "xhtml"}
	for i := 0; i < 100; i++ {
		res, err := template.New("title.xhtml").Parse(`{{template "xhtml" .}}`)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range all {
			_, err := res.New(name).Parse(inlined[name])
			if err != nil {
				t.Fatal(err)
			}
		}
		var buf bytes.Buffer
		res.Execute(&buf, 0)
		if buf.String() != "stylesheet" {
			t.Fatalf("iteration %d: got %q; expected %q", i, buf.String(), "stylesheet")
		}
	}
}
