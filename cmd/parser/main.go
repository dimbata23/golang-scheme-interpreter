package main

import (
	"fmt"
	"io/ioutil"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/parser"
)

func main() {
	str, err := ioutil.ReadFile("test/testfile.scm")
	if err == nil {
		p := parser.Parse(string(str))
		fmt.Println("Parsing...")
		for {
			expr := p.Next()
			if expr == nil {
				break
			}

			fmt.Println(expr)
		}
		fmt.Println("Done.")
	} else {
		fmt.Println(err.Error())
	}
}
