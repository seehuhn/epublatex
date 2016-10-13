// macros_test.go -
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

package tokenizer

import (
	"bytes"
	"testing"
)

func TestReadMacroName(t *testing.T) {
	testCases := []struct{ in, out string }{
		{"\\test", "\\test"},
		{"\\test o'clock", "\\test"},
		{"\\test4testing", "\\test"},
		{"\\t2", "\\t"},
		{"\\2t", "\\2"},
		{"\\{}", "\\{"},
		{"\\...", "\\."},
	}
	for i, testCase := range testCases {
		p := NewTokenizer()
		p.Prepend([]byte(testCase.in), "test data")
		p.Next()
		res, err := p.readMacroName()
		if err != nil {
			t.Error("failed to read macro name", err)
		} else if res != testCase.out {
			t.Errorf("test %d: wrong macro name, expected %q, got %q",
				i, testCase.out, res)
		}
	}
}

func TestReadOptionalArg(t *testing.T) {
	testCases := []struct {
		text     string
		expected string
		next     string
	}{
		{"hello", "", "hello"},
		{"{hello}", "", "{hello}"},
		{" hello", "", " hello"},
		{" {hello}", "", " {hello}"},
		{"[abc]def", "abc", "def"},
		{"[abc]{def}", "abc", "{def}"},
		{"[abc] {def}", "abc", " {def}"},
		{" [abc]def", "abc", "def"},
		{" [abc]{def}", "abc", "{def}"},
		{" [abc] {def}", "abc", " {def}"},
	}
	for i, testCase := range testCases {
		p := NewTokenizer()
		p.Prepend([]byte(testCase.text), "test data")
		arg, err := p.readOptionalArg()

		expected := testCase.expected
		if arg != expected {
			t.Fatal(i, "wrong arg", arg, "vs.", expected)
		}

		if !p.Next() {
			t.Fatal(i, "unexpected EOF")
		}
		buf, err := p.Peek()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(buf, []byte(testCase.next)) {
			t.Error(i, "wrong text", string(buf))
		}
	}
}

func TestReadAllMacroArgs(t *testing.T) {
	testCases := []struct {
		text     string
		expected []string
		next     string
	}{
		{"hello", nil, "hello"},
		{" hello", nil, " hello"},
		{"{a}b", []string{"a"}, "b"},
		{"{a}{b}c", []string{"a", "b"}, "c"},
		{"[a]{b}c", []string{"a", "b"}, "c"},
		{"{a}[b]c", []string{"a", "b"}, "c"},
		{"{a}%\n {b}c", []string{"a", "b"}, "c"},
	}
	for i, testCase := range testCases {
		p := NewTokenizer()
		p.Prepend([]byte(testCase.text), "test data")
		args, err := p.readAllMacroArgs()

		expected := testCase.expected
		if len(args) != len(expected) {
			t.Fatal(i, "wrong number of args", len(args), "vs.", len(expected))
		}
		// for j, arg := range args {
		//	if arg != expected[j] {
		//		t.Error(i, "wrong arg", j, "got", arg, "!=", expected[j])
		//	}
		// }

		if !p.Next() {
			t.Fatal(i, "unexpected EOF")
		}
		buf, err := p.Peek()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(buf, []byte(testCase.next)) {
			t.Error(i, "wrong text", string(buf))
		}
	}
}

func TestSubstituteMacroArgs(t *testing.T) {
	testCases := []struct {
		body     string
		args     []string
		expected string
	}{
		{" abc ", nil, " abc "},
		{"xxx#1zzz", []string{"yyy"}, "xxxyyyzzz"},
		{"#1#2#3###5", []string{"1", "2", "3", "4", "5"}, "123#5"},
	}

	for i, testCase := range testCases {
		got := substituteMacroArgs(testCase.body, testCase.args)
		if got != testCase.expected {
			t.Error("test case", i, "failed, got", got, "expected",
				testCase.expected)
		}
	}
}

func TestParseVerb(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{"|hello| and", "hello"},
		{"/hello/ and", "hello"},
		{"|}\\test{| and", "}\\test{"},
	}
	for i, testCase := range testCases {
		p := NewTokenizer()
		p.Prepend([]byte(testCase.in), "test input")

		toks, err := parseVerb(p, "\\verb")
		if err != nil {
			t.Error("test case", i, "got error", err)
			continue
		}
		if len(toks) != 1 {
			t.Error("test case", i, "got len(toks) =", len(toks))
			continue
		}
		tok := toks[0]
		if tok.Type != TokenMacro {
			t.Error("test case", i, "got wrong token type", tok.Type)
			continue
		}
		if tok.Name != "\\verb" {
			t.Error("test case", i, "got wrong token name", tok.Name)
			continue
		}
		if len(tok.Args) != 1 {
			t.Error("test case", i, "got wrong number of arguments:",
				len(tok.Args))
			continue
		}
		args := tok.Args[0].Value
		if len(args) != 1 {
			t.Error("test case", i, "got wrong length of argument:",
				len(args))
			continue
		}
		arg := args[0]
		if arg.Type != TokenVerbatim {
			t.Error("test case", i, "arg got wrong token type", tok.Type)
			continue
		}
		if arg.Name != testCase.out {
			t.Error("test case", i, "got wrong argument: expected",
				testCase.out, "got", arg.Name)
			continue
		}

		if !p.Next() {
			t.Error("test case", i, "got unexpected EOF")
			continue
		}
		buf, err := p.Peek()
		if err != nil {
			t.Error("test case", i, "got error", err)
			continue
		}

		if string(buf) != " and" {
			t.Error("test case", i, "got wrong continuation", string(buf))
		}
	}
}
