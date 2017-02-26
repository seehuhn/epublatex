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
	"path"
	"strings"
	"text/template"
)

//go:generate go run ./embed.go ../tmpl/

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

func loadTemplates(names []string) (*template.Template, error) {
	var res *template.Template

	for key := range templateFiles {
		if strings.HasPrefix(key, "parts/") {
			names = append(names, key)
		}
	}

	for _, name := range names {
		var tmpl *template.Template
		baseName := path.Base(name)
		if res == nil {
			tmpl = template.New(baseName).Funcs(templateFunctions)
			res = tmpl
		} else {
			tmpl = res.New(baseName)
		}
		_, err := tmpl.Parse(templateFiles[name])
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (w *book) writeTemplates(tmplFiles []string, data interface{}) error {
	tmpl, err := loadTemplates(tmplFiles)
	if err != nil {
		return err
	}
	return tmpl.Execute(w.current, map[string]interface{}{
		"This": data,
		"Book": w,
	})
}

func (w *book) addFileFromTemplate(path string, tmplFiles []string,
	data interface{}) error {
	err := w.createFile(path)
	if err != nil {
		return err
	}
	err = w.writeTemplates(tmplFiles, data)
	if err != nil {
		return err
	}
	return w.closeFile()
}
