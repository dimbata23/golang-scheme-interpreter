package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/interpreter"
)

func main() {
	i := interpreter.MakeInterpreter()
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("ERR READING: %s\n", err)
		}

		status := i.Interpret(input)
		if status != interpreter.StatusOk {
			break
		}
	}
}
