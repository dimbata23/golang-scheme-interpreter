package main

import (
	"fmt"
	"io/ioutil"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/lexer"
)

func main() {
	str, err := ioutil.ReadFile("test/testfile.scm")
	if err == nil {
		l := lexer.NewLexer(string(str))
		fmt.Println("Lexing...")
		for {
			token := l.NextToken()
			if token == nil {
				break
			}

			fmt.Println(token)
		}
		fmt.Println("Done.")
	} else {
		fmt.Println(err.Error())
	}
}
