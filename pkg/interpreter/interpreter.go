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
			case "if":
				return env.evalIf(ex)
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

func (env *environment) evalIf(lst *p.ExprList) p.Expression {
	cond := env.eval(lst.Lst[1])
	if cond == nil {
		fmt.Printf("DEBUG: unknown %q\n", lst.Lst[1].String())
		return nil // TODO: err?
	}

	if len(lst.Lst) < 3 || len(lst.Lst) > 4 {
		fmt.Printf("DEBUG: bad syntax: if expects 2 or 3 arguments\n")
		return nil // TODO: err?
	}

	if p.IsFalseSym(cond) {
		// false case
		if len(lst.Lst) == 4 {
			return env.eval(lst.Lst[3])
		}

		return nil // TODO: void type
	}

	// true case
	return env.eval(lst.Lst[2])
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
		return proc.Fn(&args)
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
		return env.parent.find(val)
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

func procSubDiv(args *p.ExprList, isSub bool) p.Expression {
	res := 0.0
	if !isSub {
		res = 1
	}

	if len(args.Lst) == 0 {
		return &p.Number{Val: res}
	}

	fnum, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		println("DEBUG: wrong argument type, only numbers expected")
		return nil // TODO:
	}
	res = fnum.Val

	if len(args.Lst) == 1 {
		if isSub {
			return &p.Number{Val: -res}
		} else {
			return &p.Number{Val: 1 / res}
		}
	}

	for _, ex := range args.Lst[1:] {
		num, isNum := ex.(*p.Number)
		if !isNum {
			println("DEBUG: wrong argument type, only numbers expected")
			return nil // TODO:
		}
		if isSub {
			res -= num.Val
		} else {
			res /= num.Val
		}
	}

	return &p.Number{Val: res}
}

func procSubtract(args *p.ExprList) p.Expression {
	return procSubDiv(args, true)
}

func procDivide(args *p.ExprList) p.Expression {
	return procSubDiv(args, false)
}

func procAddMult(args *p.ExprList, isAdd bool) p.Expression {
	res := 0.0
	if !isAdd {
		res = 1
	}

	for _, ex := range args.Lst {
		num, isNum := ex.(*p.Number)
		if !isNum {
			println("DEBUG: wrong argument type, only numbers expected")
			return nil // TODO:
		}

		if isAdd {
			res += num.Val
		} else {
			res *= num.Val
		}
	}

	return &p.Number{Val: res}
}

func procAdd(args *p.ExprList) p.Expression {
	return procAddMult(args, true)
}

func procMultiply(args *p.ExprList) p.Expression {
	return procAddMult(args, false)
}

func procComp(args *p.ExprList, comp func(*p.Number, *p.Number) bool) p.Expression {
	if len(args.Lst) == 0 {
		return &p.TrueSym
	}

	lastNum, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String())
		return nil // TODO:
	}

	for _, ex := range args.Lst[1:] {
		num, isNum := ex.(*p.Number)
		if !isNum {
			fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String())
			return nil // TODO:
		}

		if !comp(lastNum, num) {
			return &p.FalseSym
		}

		lastNum = num
	}

	return &p.TrueSym
}

func less(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val < rhs.Val
}

func lessEq(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val <= rhs.Val
}

func greater(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val > rhs.Val
}

func greaterEq(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val >= rhs.Val
}

func equal(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val == rhs.Val
}

func procLess(args *p.ExprList) p.Expression {
	return procComp(args, less)
}

func procLessEq(args *p.ExprList) p.Expression {
	return procComp(args, lessEq)
}

func procGreater(args *p.ExprList) p.Expression {
	return procComp(args, greater)
}

func procGreaterEq(args *p.ExprList) p.Expression {
	return procComp(args, greaterEq)
}

func procEquals(args *p.ExprList) p.Expression {
	return procComp(args, equal)
}

type interpreter struct {
	genv environment
}

func (i *interpreter) addDefaultDefs() *interpreter {
	i.genv.parent = nil
	i.genv.vars = map[string]p.Expression{
		"#f": &p.FalseSym,
		"#t": &p.TrueSym,
		"+":  &p.Procedure{Fn: procAdd},
		"*":  &p.Procedure{Fn: procMultiply},
		"-":  &p.Procedure{Fn: procSubtract},
		"/":  &p.Procedure{Fn: procDivide},
		"=":  &p.Procedure{Fn: procEquals},
		"<":  &p.Procedure{Fn: procLess},
		"<=": &p.Procedure{Fn: procLessEq},
		">":  &p.Procedure{Fn: procGreater},
		">=": &p.Procedure{Fn: procGreaterEq},
		// TODO: list, cons, car, cdr, number?, null?, pair?, list?, string?, and, or, remainder, quotient, expt, display
	}

	return i
}

func NewInterpreter() *interpreter {
	res := interpreter{}
	return res.addDefaultDefs()
}

func MakeInterpreter() interpreter {
	res := interpreter{}
	return *res.addDefaultDefs()
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
