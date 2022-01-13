package interpreter

import "github.com/dimbata23/golang-scheme-interpreter/pkg/parser"

type environment struct {
	vars   map[string]parser.Expression
	parent *environment
}

func (env *environment) eval(expr parser.Expression) parser.Expression {
	switch ex := expr.(type) {

	case *parser.Variable:
		return env.find(ex.Val)

	case *parser.Symbol:
		return ex

	case *parser.Number:
		return ex

	case *parser.Procedure:
		panic("unimplemented")
		//return env.evalProcLambda(ex)

	case *parser.ExprList:
		if len(ex.Lst) == 1 && parser.IsNullSym(ex.Lst[0]) {
			panic("shouldn't happen")
		}

		if len(ex.Lst) == 0 {
			println("Missing procedure")
			return nil // TODO: return an error with the msg?
		}

		if v, isVar := ex.Lst[0].(*parser.Variable); isVar {
			switch v.Val {
			case "define":
				return env.evalDefine(ex)
				//lambda, if, cond, apply, map, quote, begin, .. ?
			default:
				panic("unimplemented")
				//return env.evalProcLambda(ex)
			}
		} else {
			panic("unimplemented")
			//return env.evalProcLambda(ex)
		}

	default:
		return nil

	}
}

func (env *environment) find(val string) parser.Expression {
	if val, ok := env.vars[val]; ok {
		return val
	}

	if env.parent != nil {
		if val, ok := env.parent.vars[val]; ok {
			return val
		}
	}

	return nil
}

func (env *environment) evalDefine(lst *parser.ExprList) parser.Expression {
	len := len(lst.Lst)

	if len < 3 {
		println("bad syntax: define needs at least 2 arguments")
		return nil
	}

	if len > 3 {
		if _, isLst := lst.Lst[1].(*parser.ExprList); !isLst {
			println("bad syntax: define expects exactly one expression after identifier")
			return nil
		}
	}

	var res parser.Expression
	var ident string

	switch firstArg := lst.Lst[1].(type) {
	case *parser.ExprList: // Lambda definition
		if lambdaName, isVar := firstArg.Lst[0].(*parser.Variable); isVar {
			ident = lambdaName.Val
			res = &parser.Lambda{Name: ident, Lst: firstArg.Lst[1:]}
		} else {
			panic("unimplemented")
		}

	case *parser.Variable: // Variable definition
		ident = firstArg.Val
		res = env.eval(lst.Lst[2])

	default:
		panic("unimplemented")
	}

	env.vars[ident] = res

	return res
}

type interpreter struct {
	genv environment
}

func (i *interpreter) addDefaultProcs() *interpreter {
	i.genv.parent = nil
	i.genv.vars = make(map[string]parser.Expression)
	return i
}

func NewInterpreter() *interpreter {
	res := interpreter{}
	return res.addDefaultProcs()
}

func MakeInterpreter() interpreter {
	res := interpreter{}
	return *res.addDefaultProcs()
}

type Status int

const (
	StatusOk Status = iota
	StatusExitted
	StatusError
)

func (i *interpreter) Interpret(input string) Status {
	p := parser.Parse(input)

	for {
		expr := p.Next()
		if expr == nil {
			//println("DEBUG: Got nil after parsing. Should appear when the Interpret() finishes.")
			//intstat = StatusOk
			break
		}

		if parser.IsSpecialExit(expr) {
			println("DEBUG: Got (exit), bye!")
			return StatusExitted
		}

		var res parser.Expression
		if e, isErr := expr.(*parser.Error); isErr {
			res = e
		} else {
			res = i.genv.eval(expr)
		}

		if res == nil {
			//println("DEBUG: Got nil after evaluating")
			//intstat = StatusError
		} else {
			println(res.String())
		}
	}

	return StatusOk
}
