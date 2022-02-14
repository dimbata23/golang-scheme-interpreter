package main

import (
	"fmt"
	"io/ioutil"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/parser"
)

func main() {
	str, err := ioutil.ReadFile("test/testfile.scm")
	if err == nil {
		p := parser.NewParser(string(str))
		fmt.Println("Parsing...")
		for {
			expr, err := p.Next()
			if expr == nil {
				break
			}

			if err != nil {
				fmt.Println(err.String())
			} else {
				fmt.Println(expr.String(0))
			}
		}
		fmt.Println("Done.")
	} else {
		fmt.Println(err.Error())
	}
}
