// +build ignore

// Tokenize input (either given on the command line or in a file)
package main

import (
	"flag"
	"fmt"

	"github.com/seehuhn/epublatex/latex/tokenizer"
)

var input = flag.String("input", "", "input to parse")

func main() {
	flag.Parse()

	c := make(chan *tokenizer.Token, 64)

	go func() {
		p := tokenizer.NewTokenizer()
		if *input != "" {
			p.Prepend([]byte(*input), "input")
		}
		err := p.ParseTex(c)
		if err != nil {
			panic(err)
		}

		for _, fname := range flag.Args() {
			err = p.Include(fname)
			if err != nil {
				panic(err)
			}
			err := p.ParseTex(c)
			if err != nil {
				panic(err)
			}
		}
		close(c)
	}()

	for tok := range c {
		fmt.Println(tok)
	}
}
