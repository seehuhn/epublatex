// template.go -
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
	"io"
	"path/filepath"
	"strings"
	"text/template"
)

func templateFormatList(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}

	var parts []string
	for i, part := range list {
		parts = append(parts, part)
		if i < len(list)-2 {
			parts = append(parts, ", ")
		} else if i < len(list)-1 {
			parts = append(parts, " and ")
		}
	}
	return strings.Join(parts, "")
}

var templateFunctions = template.FuncMap{
	"formatlist": templateFormatList,
}

func (w *epub) writeTemplates(out io.Writer, tmplFiles []string,
	data interface{}) error {

	tmp := make([]string, len(tmplFiles))
	for i, f := range tmplFiles {
		tmp[i] = filepath.Join(w.tmplDir, f)
	}
	tmplFiles = tmp

	name := filepath.Base(tmplFiles[0])
	tmpl, err := template.New(name).
		Funcs(templateFunctions).
		ParseFiles(tmplFiles...)
	if err != nil {
		return err
	}
	// TODO(voss): don't hardcode the parts directory
	tmpl, err = tmpl.ParseGlob(filepath.Join(w.tmplDir, "parts", "*"))
	if err != nil {
		return err
	}
	return tmpl.Execute(out, map[string]interface{}{
		"This": data,
		"Book": w,
	})
}

func (w *epub) addFileFromTemplate(path string, tmplFiles []string,
	data interface{}) error {
	err := w.createFile(path)
	if err != nil {
		return err
	}
	err = w.writeTemplates(w.current, tmplFiles, data)
	if err != nil {
		return err
	}
	return w.closeFile()
}
