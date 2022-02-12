package interpreter

import (
	"fmt"
	"io/ioutil"
	"math"

	p "github.com/dimbata23/golang-scheme-interpreter/pkg/parser"
)

type environment struct {
	vars   map[string]p.Expression
	parent *environment
}

func (env *environment) eval(expr p.Expression) p.Expression {
	switch ex := expr.(type) {

	case *p.Variable:
		if res := env.find(ex.Val); res != nil {
			return res
		}

		fmt.Printf("DEBUG: Unbound variable: %q\n", ex.String(0))
		return nil // TODO:

	case *p.Symbol:
		return ex

	case *p.Number:
		return ex

	case *p.Procedure:
		panic("unreachable")
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
			case "if":
				return env.evalIf(ex)
			case "load":
				return env.evalLoad(ex)
				//lambda, cond, apply, map, quote, begin, .. ?
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
			fmt.Printf("DEBUG: non-variable param given %q\n", param.String(0))
		}
	}

	return resEnv
}

func (env *environment) evalIf(lst *p.ExprList) p.Expression {
	if len(lst.Lst) < 3 || len(lst.Lst) > 4 {
		fmt.Printf("DEBUG: bad syntax: if expects 2 or 3 arguments\n")
		return nil // TODO: err?
	}

	cond := env.eval(lst.Lst[1])
	if cond == nil {
		fmt.Printf("DEBUG: unknown %q\n", lst.Lst[1].String(0))
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

func (env *environment) evalLoad(lst *p.ExprList) p.Expression {
	if len(lst.Lst) != 2 {
		fmt.Printf("DEBUG: bad syntax: load expects 1 argument\n")
		return nil // TODO: err?
	}

	arg := lst.Lst[1]
	if fileName, isVar := arg.(*p.Variable); isVar {
		input, err := ioutil.ReadFile(fileName.Val)
		if err != nil {
			fmt.Printf("DEBUG: there was an error loading the file %q\n", fileName.Val)
			return nil
		}

		var res p.Expression = nil
		par := p.Parse(string(input))
		for {
			expr := par.Next()
			if expr == nil {
				break // parser has finished
			}

			if e, isErr := expr.(*p.Error); isErr {
				res = e
			} else {
				res = env.eval(expr)
			}

			if res != nil {
				println(res.String(0))
			}
		}
	}

	return nil // TODO: void?
}

func (env *environment) evalProcLambda(lst *p.ExprList) p.Expression {
	pr := env.eval(lst.Lst[0])
	if pr == nil {
		fmt.Printf("DEBUG: unknown %q\n", lst.Lst[0].String(0))
		return nil // TODO: err?
	}

	proc, isProc := pr.(*p.Procedure)
	lambda, isLambda := pr.(*p.Lambda)

	if !isProc && !isLambda {
		fmt.Printf("DEBUG: %q not a procedure\n", pr.String(0))
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
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
		return nil // TODO:
	}

	for _, ex := range args.Lst[1:] {
		num, isNum := ex.(*p.Number)
		if !isNum {
			fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
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

func procIsNumber(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch\n")
		return nil // TODO:
	}

	if _, isNum := args.Lst[0].(*p.Number); isNum {
		return &p.TrueSym
	}

	return &p.FalseSym
}

func procIsNull(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch\n")
		return nil // TODO:
	}

	if p.IsNullSym(args.Lst[0]) {
		return &p.TrueSym
	}

	return &p.FalseSym
}

func procAnd(args *p.ExprList) p.Expression {
	res := p.Expression(&p.TrueSym)

	for _, ex := range args.Lst {
		if p.IsFalseSym(ex) {
			return &p.FalseSym
		}

		res = ex
	}

	return res
}

func procOr(args *p.ExprList) p.Expression {
	res := p.Expression(&p.FalseSym)

	for _, ex := range args.Lst {
		if !p.IsFalseSym(ex) {
			return ex
		}
	}

	return res
}

func procRemainder(args *p.ExprList) p.Expression {
	if len(args.Lst) != 2 {
		fmt.Printf("DEBUG: arity mismatch, expected 2\n")
		return nil // TODO:
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
		return nil // TODO:
	}

	div, isNumDiv := args.Lst[1].(*p.Number)
	if !isNumDiv {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[1].String(0))
		return nil // TODO:
	}

	return &p.Number{Val: float64(int64(num.Val) % int64(div.Val))}
}

func procQuotient(args *p.ExprList) p.Expression {
	if len(args.Lst) != 2 {
		fmt.Printf("DEBUG: arity mismatch, expected 2\n")
		return nil // TODO:
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
		return nil // TODO:
	}

	div, isNumDiv := args.Lst[1].(*p.Number)
	if !isNumDiv {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[1].String(0))
		return nil // TODO:
	}

	return &p.Number{Val: float64(int64(num.Val) / int64(div.Val))}
}

func procExpt(args *p.ExprList) p.Expression {
	if len(args.Lst) != 2 {
		fmt.Printf("DEBUG: arity mismatch, expected 2\n")
		return nil // TODO:
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
		return nil // TODO:
	}

	exp, isExpDiv := args.Lst[1].(*p.Number)
	if !isExpDiv {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[1].String(0))
		return nil // TODO:
	}

	return &p.Number{Val: math.Pow(num.Val, exp.Val)}
}

func procList(args *p.ExprList) p.Expression {
	if len(args.Lst) == 0 {
		return &p.NullSym
	}

	args.Lst = append(args.Lst, &p.NullSym)

	return args
}

func procCons(args *p.ExprList) p.Expression {
	if len(args.Lst) != 2 {
		fmt.Printf("DEBUG: arity mismatch, expected 2\n")
		return nil // TODO:
	}

	resLst := args.Lst[0:2]
	if secArg, isLst := args.Lst[1].(*p.ExprList); isLst && secArg.Qlevel <= 1 {
		resLst = append(args.Lst[0:1], secArg.Lst...)
	}

	return &p.ExprList{Lst: resLst, Qlevel: 1}
}

func procCar(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch, expected 1\n")
		return nil // TODO:
	}

	arg := args.Lst[0]
	if lstArg, isLst := arg.(*p.ExprList); isLst {
		return lstArg.Lst[0]
	}

	fmt.Printf("DEBUG: Contract vialotion, expected pair?, got %q\n", arg.String(0))
	return nil
}

func procCdr(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch, expected 1\n")
		return nil // TODO:
	}

	arg := args.Lst[0]
	if pairArg, isPair := isPair(arg); isPair {
		if len(pairArg.Lst) == 2 {
			return pairArg.Lst[1]
		}

		return &p.ExprList{Lst: pairArg.Lst[1:], Qlevel: pairArg.Qlevel}
	}

	fmt.Printf("DEBUG: Contract vialotion, expected pair?, got %q\n", arg.String(0))
	return nil
}

func isPair(arg p.Expression) (pair *p.ExprList, isPair bool) {
	if pair, isPair := arg.(*p.ExprList); isPair && len(pair.Lst) >= 2 {
		return pair, isPair
	}

	return nil, false
}

func procIsList(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch, expected 1\n")
		return nil // TODO:
	}

	arg := args.Lst[0]
	if p.IsNullSym(arg) {
		return &p.TrueSym
	}

	if lst, isList := arg.(*p.ExprList); isList {
		len := len(lst.Lst)
		if len == 0 || p.IsNullSym(lst.Lst[len-1]) {
			return &p.TrueSym
		}
	}

	return &p.FalseSym
}

func procIsPair(args *p.ExprList) p.Expression {
	if len(args.Lst) != 1 {
		fmt.Printf("DEBUG: arity mismatch, expected 1\n")
		return nil // TODO:
	}

	if _, isPair := isPair(args.Lst[0]); isPair {
		return &p.TrueSym
	}

	return &p.FalseSym
}

func minMax(args *p.ExprList) (min *p.Number, max *p.Number) {
	if len(args.Lst) == 0 {
		fmt.Printf("DEBUG: arity mismatch, expected at least 1\n")
		return nil, nil // TODO:
	}

	max, isNum := args.Lst[0].(*p.Number)
	min, _ = args.Lst[0].(*p.Number)
	if !isNum {
		fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", args.Lst[0].String(0))
		return nil, nil // TODO:
	}

	for _, expr := range args.Lst[1:] {
		if curr, isNum := expr.(*p.Number); isNum {
			if curr.Val > max.Val {
				max = curr
			}
			if curr.Val < min.Val {
				min = curr
			}
		} else {
			fmt.Printf("DEBUG: Contract vialotion, expected number?, got %q\n", expr.String(0))
			return nil, nil // TODO:
		}
	}

	return min, max
}

func procMax(args *p.ExprList) p.Expression {
	_, max := minMax(args)
	if max == nil {
		return nil // because interfaces can have a type with a nil value... (:
	}

	return max // why golang... why...
}

func procMin(args *p.ExprList) p.Expression {
	min, _ := minMax(args)
	if min == nil {
		return nil // because interfaces can have a type with a nil value... (:
	}

	return min // why golang... why...
}

type interpreter struct {
	genv environment
}

func (i *interpreter) addDefaultDefs() *interpreter {
	i.genv.parent = nil
	i.genv.vars = map[string]p.Expression{
		"#f":        &p.FalseSym,
		"#t":        &p.TrueSym,
		"+":         &p.Procedure{Fn: procAdd},
		"*":         &p.Procedure{Fn: procMultiply},
		"-":         &p.Procedure{Fn: procSubtract},
		"/":         &p.Procedure{Fn: procDivide},
		"=":         &p.Procedure{Fn: procEquals},
		"<":         &p.Procedure{Fn: procLess},
		"<=":        &p.Procedure{Fn: procLessEq},
		">":         &p.Procedure{Fn: procGreater},
		">=":        &p.Procedure{Fn: procGreaterEq},
		"number?":   &p.Procedure{Fn: procIsNumber},
		"null?":     &p.Procedure{Fn: procIsNull},
		"and":       &p.Procedure{Fn: procAnd},
		"or":        &p.Procedure{Fn: procOr},
		"remainder": &p.Procedure{Fn: procRemainder},
		"quotient":  &p.Procedure{Fn: procQuotient},
		"expt":      &p.Procedure{Fn: procExpt},
		"list":      &p.Procedure{Fn: procList},
		"cons":      &p.Procedure{Fn: procCons},
		"car":       &p.Procedure{Fn: procCar},
		"cdr":       &p.Procedure{Fn: procCdr},
		"pair?":     &p.Procedure{Fn: procIsPair},
		"list?":     &p.Procedure{Fn: procIsList},
		"max":       &p.Procedure{Fn: procMax},
		"min":       &p.Procedure{Fn: procMin},
		// TODO: string?, display
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

		if res != nil {
			println(res.String(0))
		}
	}

	return StatusOk
}
