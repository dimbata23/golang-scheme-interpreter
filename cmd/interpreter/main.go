package main

import (
	"fmt"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/interpreter"
)

func main() {
	i := interpreter.MakeInterpreter()
	for {
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println(err)
		}

		status := i.Interpret(input)
		if status != interpreter.StatusOk {
			break
		}
	}
}
