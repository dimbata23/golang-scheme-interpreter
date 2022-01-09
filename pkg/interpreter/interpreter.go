package interpreter

import "github.com/dimbata23/golang-scheme-interpreter/pkg/parser"

type environment struct {
	vars   map[string]parser.Expression
	parent *environment
}

func (env *environment) eval(expr parser.Expression) parser.Expression {
	panic("not implemented")
}

type interpreter struct {
	genv environment
}

func NewInterpreter() *interpreter {
	// TODO: Create default procedures
	return &interpreter{}
}

func MakeInterpreter() interpreter {
	// TODO: Create default procedures
	return interpreter{}
}

type Status int

const (
	StatusOk Status = iota
	StatusExitted
	StatusError
)

func (i *interpreter) Interpret(input string) (intstat Status) {
	p := parser.Parse(input)
	for {
		expr := p.Next()
		if expr == nil {
			println("DEBUG: Got nil after parsing. Should appear when the Interpret() finishes.")
			intstat = StatusOk
			break
		}

		if parser.IsSpecialExit(expr) {
			println("DEBUG: Got (exit), bye!")
			intstat = StatusExitted
			break
		}

		res := i.genv.eval(expr)

		if res == nil {
			println("DEBUG: Got nil after evaluating")
			intstat = StatusError
			break
		}

		println(res)
	}

	return intstat
}
