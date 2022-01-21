package interpreter

import (
	"fmt"

	p "github.com/dimbata23/golang-scheme-interpreter/pkg/parser"
)

type environment struct {
	vars   map[string]p.Expression
	parent *environment
}

func (env *environment) eval(expr p.Expression) p.Expression {
	switch ex := expr.(type) {

	case *p.Variable:
		return env.find(ex.Val)

	case *p.Symbol:
		return ex

	case *p.Number:
		return ex

	case *p.Procedure:
		panic("unimplemented")
		//return env.evalProcLambda(ex)

	case *p.ExprList:
		if ex.Qlevel > 0 {
			return ex
		}

		if len(ex.Lst) == 0 {
			println("DEBUG: Missing procedure")
			return nil // TODO: return an error with the msg?
		}

		if len(ex.Lst) == 1 && p.IsNullSym(ex.Lst[0]) {
			panic("shouldn't happen")
		}

		// Special forms
		if v, isVar := ex.Lst[0].(*p.Variable); isVar {
			switch v.Val {
			case "define":
				return env.evalDefine(ex)
				//lambda, if, cond, apply, map, quote, begin, .. ?
			}
		}

		// Non-special form
		return env.evalProcLambda(ex)

	default:
		return nil

	}
}

func makeEnvironment(parent *environment, params *p.ExprList, args *p.ExprList) environment {
	resEnv := environment{parent: parent, vars: make(map[string]p.Expression, len(params.Lst))}

	for i, param := range params.Lst {
		if vp, isVar := param.(*p.Variable); isVar {
			resEnv.vars[vp.Val] = args.Lst[i]
		} else {
			fmt.Printf("DEBUG: non-variable param given %q\n", param.String())
		}
	}

	return resEnv
}

func (env *environment) evalProcLambda(lst *p.ExprList) p.Expression {
	pr := env.eval(lst.Lst[0])
	if pr == nil {
		fmt.Printf("DEBUG: unknown %q\n", lst.Lst[0].String())
		return nil // TODO: err?
	}

	proc, isProc := pr.(*p.Procedure)
	lambda, isLambda := pr.(*p.Lambda)

	if !isProc && !isLambda {
		fmt.Printf("DEBUG: %q not a procedure\n", pr.String())
		return nil // TODO:
	}

	argsLen := len(lst.Lst[1:])
	args := p.ExprList{Lst: make([]interface{ p.Expression }, argsLen)}

	for i, arg := range lst.Lst[1:] {
		args.Lst[i] = env.eval(arg) // TODO: concurency/parallelism
		if args.Lst[i] == nil {
			println("DEBUG: something broke while evaluating proc/lambda arguments")
			return nil
		}
	}

	if isProc {
		proc.Fn(&args)
	} else if isLambda {
		if len(lambda.Params.Lst) != len(args.Lst) {
			println("DEBUG: arity mismatch")
			return nil // TODO:
		}

		var res p.Expression
		lambdaEnv := makeEnvironment(env, lambda.Params, &args)
		for _, expr := range lambda.Body.Lst {
			res = lambdaEnv.eval(expr)
			if res == nil {
				println("DEBUG: something broke while evaluating lambda body")
				return nil // TODO:
			}
		}

		return res
	}

	panic("unreachable")
}

func (env *environment) find(val string) p.Expression {
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

func (env *environment) evalDefine(lst *p.ExprList) p.Expression {
	len := len(lst.Lst)

	if len < 3 {
		println("DEBUG: bad syntax: define needs at least 2 arguments")
		return nil
	}

	if len > 3 {
		if _, isLst := lst.Lst[1].(*p.ExprList); !isLst {
			println("DEBUG: bad syntax: define expects exactly one expression after identifier")
			return nil
		}
	}

	var res p.Expression
	var ident string

	switch firstArg := lst.Lst[1].(type) {
	case *p.ExprList: // Lambda definition
		if lambdaName, isVar := firstArg.Lst[0].(*p.Variable); isVar {
			ident = lambdaName.Val
			params := p.ExprList{Lst: firstArg.Lst[1:]}
			body := p.ExprList{Lst: lst.Lst[2:]}
			res = &p.Lambda{Name: ident, Params: &params, Body: &body}
		} else {
			println("DEBUG: lambda definition name not of variable type")
			return nil // TODO:
		}

	case *p.Variable: // Variable definition
		ident = firstArg.Val
		res = env.eval(lst.Lst[2])

	default:
		println("DEBUG: wrong argument type")
		return nil // TODO:
	}

	env.vars[ident] = res

	return res
}

type interpreter struct {
	genv environment
}

func (i *interpreter) addDefaultProcs() *interpreter {
	i.genv.parent = nil
	i.genv.vars = map[string]p.Expression{}

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
	par := p.Parse(input)

	for {
		expr := par.Next()
		if expr == nil {
			//println("DEBUG: Got nil after parsing. Should appear when the Interpret() finishes.")
			//intstat = StatusOk
			break
		}

		if p.IsSpecialExit(expr) {
			println("DEBUG: Got (exit), bye!")
			return StatusExitted
		}

		var res p.Expression
		if e, isErr := expr.(*p.Error); isErr {
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
