package main

import (
	"fmt"
	"io/ioutil"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/lexer"
)

func main() {
	str, err := ioutil.ReadFile("test/testfile.scm")
	if err == nil {
		l := lexer.Lex(string(str))
		fmt.Println("Lexing...")
		for {
			item := l.NextItem()
			if item == nil {
				break
			}

			fmt.Println(item)
		}
		fmt.Println("Done.")
	} else {
		fmt.Println(err.Error())
	}
}
