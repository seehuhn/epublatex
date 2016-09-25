// macros.go -
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
	"html"
	"log"
	"strconv"

	"github.com/seehuhn/epublatex/latex/tokenizer"
)

var pkgInit = map[string]func(conv *converter, options string){
	"amsmath":  addAmsmathMacros,
	"amsfonts": addNoMacros,
	"amsthm":   addAmsthmMacros,
}

type macro interface {
	HTMLOutput(args []*tokenizer.Arg, conv *converter) string
}

func (conv *converter) addBuiltinMacros() {
	// builtin EPUB support
	conv.Macros["\\epubauthor"] = funcMacro(mEpubAuthor)
	conv.Macros["\\epubtitle"] = funcMacro(mEpubTitle)

	// TeX/LaTeX macros
	conv.Macros["\\documentclass"] = mIgnore
	conv.Macros["\\label"] = mIgnore // handled during pass 1
	conv.Macros["\\ref"] = funcMacro(mRef)
	conv.Macros["\\usepackage"] = funcMacro(mUsePackage)
	for name, out := range map[string]string{
		"\\textit": "i",
		"\\textbf": "b",
	} {
		conv.Macros[name] = mHTMLTag(out)
	}
	for name, out := range map[string]string{
		"\\dots": horizonalEllipsis,
	} {
		conv.Macros[name] = mSubst(out)
	}

	conv.Counters["base@equation"] = &counterInfo{}
	conv.Envs["equation"] = &environment{
		Prefix:  "Equation",
		Counter: "base@equation",
	}
}

func addNoMacros(conv *converter, options string) {}

func mEpubAuthor(args []*tokenizer.Arg, conv *converter) string {
	conv.Author = args[0].String()
	return ""
}

func mEpubTitle(args []*tokenizer.Arg, conv *converter) string {
	conv.Title = args[0].String()
	return ""
}

func mRef(args []*tokenizer.Arg, conv *converter) string {
	target := args[0].String()
	for _, label := range conv.Labels {
		if label.Label == target {
			// TODO(voss): make this more robust
			fname := "ch" + strconv.Itoa(label.Chapter) + ".xhtml"
			return `<a href="` + fname + `#` + label.ID + `">` + label.Name + `</a>`
		}
	}
	return `<span class="error">` + html.EscapeString(target) + `</span>`
}

func mUsePackage(args []*tokenizer.Arg, conv *converter) string {
	options := args[0].String()
	pkgName := args[1].String()
	installFn := pkgInit[pkgName]
	if installFn != nil {
		installFn(conv, options)
	} else {
		log.Printf("unknown package %q (options %q)", pkgName, options)
	}
	return ""
}

type mIgnoreClass struct{}

func (m mIgnoreClass) HTMLOutput(args []*tokenizer.Arg, conv *converter) string {
	return ""
}

var mIgnore = mIgnoreClass{}

type mSubst string

func (m mSubst) HTMLOutput(args []*tokenizer.Arg, conv *converter) string {
	return string(m)
}

type mHTMLTag string

func (m mHTMLTag) HTMLOutput(args []*tokenizer.Arg, conv *converter) string {
	startTag := "<" + string(m) + ">"
	endTag := "</" + string(m) + ">"
	return startTag + conv.convertHTML(args[0].Value) + endTag
}

type funcMacro func(args []*tokenizer.Arg, conv *converter) string

func (m funcMacro) HTMLOutput(args []*tokenizer.Arg, conv *converter) string {
	return m(args, conv)
}
