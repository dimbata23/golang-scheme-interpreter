package interpreter

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"

	p "github.com/dimbata23/golang-scheme-interpreter/pkg/parser"
)

/// ------------------------------------------------------------------------ ///
/// -------------------------- Public definitions -------------------------- ///
/// ------------------------------------------------------------------------ ///

// the interpreter struct
type Interpreter struct {
	genv environment
}

type Status int // status of the interpreter after interpreting

const (
	StatusOk      Status = iota // interpreting finished successfully
	StatusExitted               // interpreter was given an exit command
	StatusError                 // an error occured while interpreting
)

/// ------------------------------------------------------------------------ ///
/// --------------------- Public functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// creates a new interpreter
func NewInterpreter() *Interpreter {
	res := Interpreter{}
	return res.addDefaultDefs()
}

// makes a new interpreter
func MakeInterpreter() Interpreter {
	res := Interpreter{}
	return *res.addDefaultDefs()
}

// interprets the given string printing any results to the console
// returns the status of the interpreter afterwards
func (i *Interpreter) Interpret(input string) Status {
	par := p.NewParser(input)

	for {
		expr, err := par.Next()
		if expr == nil {
			break // parser has finished
		}

		if p.IsSpecialExit(expr) {
			fmt.Println("Got (exit), bye!")
			return StatusExitted
		}

		if err == nil {
			expr, err = i.genv.eval(expr)
		}

		if err != nil {
			fmt.Println(err.String())
		} else {
			fmt.Println(expr.String(0))
		}
	}

	return StatusOk
}

/// ------------------------------------------------------------------------ ///
/// --------------------------- Inner structure ---------------------------- ///
/// ------------------------------------------------------------------------ ///

// environment containing definitions of expressions
type environment struct {
	vars   map[string]p.Expression // expression definitions
	parent *environment            // a parent environment, if any
}

type errorType int

const (
	errUnknown errorType = iota
	errUnboundIdentifier
	errMissingProc
	errBadSyntax
	errCouldntEval
	errCouldntLoadFile
	errNotAProc
	errArityMismatch
	errContractViolation
)

/// ------------------------------------------------------------------------ ///
/// -------------------- Private functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// creates an environment
func makeEnvironment(parent *environment, params *p.ExprList, args *p.ExprList) environment {
	resEnv := environment{
		parent: parent,
		vars:   make(map[string]p.Expression, len(params.Lst)),
	}

	for i, param := range params.Lst {
		if vp, isVar := param.(*p.Variable); isVar {
			resEnv.vars[vp.Val] = args.Lst[i]
		} else {
			fmt.Printf("DEBUG: non-variable param given %q\n", param.String(0))
		}
	}

	return resEnv
}

// evaluates the given expression
// can return an error
func (env *environment) eval(expr p.Expression) (ex p.Expression, err *p.Error) {
	switch ex := expr.(type) {

	case *p.Variable:
		return env.find(ex.Val)

	case *p.Symbol:
		return ex, nil

	case *p.Number:
		return ex, nil

	case *p.ExprList:
		if ex.Qlevel > 0 {
			return ex, nil
		}

		if len(ex.Lst) == 0 {
			return &p.Void, newError(errMissingProc)
		}

		if len(ex.Lst) == 1 && p.IsNullSym(ex.Lst[0]) {
			panic("shouldn't happen")
		}

		// special forms
		if v, isVar := ex.Lst[0].(*p.Variable); isVar {
			switch v.Val {
			case "define":
				return env.evalDefine(ex)
			case "if":
				return env.evalIf(ex)
			case "load":
				return env.evalLoad(ex)
			case "cond":
				return env.evalCond(ex)
			case "lambda":
				return env.evalLambda(ex)
			}
		}

		// lambda/procedure
		return env.evalProcLambda(ex)

	case *p.Procedure:
		panic("unreachable")

	default:
		return &p.Void, newError(errUnknown)

	}
}

// returns an expression defined by the given string
// or an error if no such definition is found
func (env *environment) find(val string) (ex p.Expression, err *p.Error) {
	if val, ok := env.vars[val]; ok {
		return val, nil
	}

	if env.parent != nil {
		return env.parent.find(val)
	}

	return &p.Void, newError(errUnboundIdentifier, val)
}

// add the default scheme definitions
func (i *Interpreter) addDefaultDefs() *Interpreter {
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
	}

	return i
}

// creates a generic error from the given type and arguments
// args can be [identifier], [expected value] and [given value] in that order
func newError(typ errorType, args ...string) (err *p.Error) {
	len := len(args)
	err = &p.Error{}

	switch typ {
	case errUnknown:
		err.Val = "an unknown error occured"

	case errUnboundIdentifier:
		err.Val = "unbound identifier"
		if len > 0 {
			err.Val = fmt.Sprintf("%s: %s", args[0], err.Val)
		}

	case errMissingProc:
		err.Val = "#%app: missing procedure expression;\n probably originally (), which is an illegal empty application"

	case errBadSyntax:
		err.Val = "bad syntax"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s: %s", args[0], err.Val)
		}

	case errCouldntEval:
		err.Val = "couldn't evaluate"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s: %s", args[0], err.Val)
		}

	case errCouldntLoadFile:
		err.Val = "load: couldn't load file"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s\n %s", err.Val, args[0])
		}

	case errNotAProc:
		err.Val = "application: not a procedure;\n expected a procedure that can be applied to arguments"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s\n  given: %s", err.Val, args[0])
		}

	case errArityMismatch:
		err.Val = "arity mismatch;\n the expected number of arguments does not match the given number"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s: %s", args[0], err.Val)
		}

	case errContractViolation:
		err.Val = "contract violation"
		if len >= 1 {
			err.Val = fmt.Sprintf("%s: %s", args[0], err.Val)
		}

	default:
		err.Val = "wrong error type"
	}

	if len >= 2 {
		err.Val = fmt.Sprintf("%s\n  expected: %s", err.Val, args[1])
	}

	if len >= 3 {
		err.Val = fmt.Sprintf("%s\n  given: %s", err.Val, args[2])
	}

	return err
}

/// ------------------------------------------------------------------------ ///
/// ------------------------- Special form methods ------------------------- ///
/// ------------------------------------------------------------------------ ///

// (if <condition> <true case> [false case])
func (env *environment) evalIf(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(lst.Lst)
	if len < 3 || len > 4 {
		return &p.Void, newError(errBadSyntax, "if", "2 or 3 arguments", strconv.Itoa(len-1))
	}

	cond, condErr := env.eval(lst.Lst[1])
	if condErr != nil {
		return &p.Void, condErr
	}

	if p.IsFalseSym(cond) {
		// false case
		if len == 4 {
			return env.eval(lst.Lst[3])
		}

		return &p.Void, nil
	}

	// true case
	return env.eval(lst.Lst[2])
}

// (load <filename>)
// interprets a scheme file
func (env *environment) evalLoad(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(lst.Lst)
	if len != 2 {
		return &p.Void, newError(errBadSyntax, "load", "1 argument", strconv.Itoa(len-1))
	}

	arg := lst.Lst[1]
	if fileName, isVar := arg.(*p.Variable); isVar {
		input, ioerr := ioutil.ReadFile(fileName.Val)
		if ioerr != nil {
			return &p.Void, newError(errCouldntLoadFile, ioerr.Error())
		}

		par := p.NewParser(string(input))
		for {
			ex, err := par.Next()
			if ex == nil {
				break // parser has finished
			}

			if err == nil {
				ex, err = env.eval(ex)
			}

			if err != nil {
				fmt.Println(err.String())
			} else {
				fmt.Println(ex.String(0))
			}
		}
	}

	return &p.Void, nil
}

// (<proc/lambda> [args...])
func (env *environment) evalProcLambda(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	pr, prErr := env.eval(lst.Lst[0])
	if prErr != nil {
		return &p.Void, prErr
	}

	proc, isProc := pr.(*p.Procedure)
	lambda, isLambda := pr.(*p.Lambda)

	if !isProc && !isLambda {
		return &p.Void, newError(errNotAProc, pr.String(0))
	}

	argsLen := len(lst.Lst[1:])
	args := p.ExprList{Lst: make([]interface{ p.Expression }, argsLen)}

	for i, arg := range lst.Lst[1:] {
		args.Lst[i], err = env.eval(arg) // TODO: concurency/parallelism
		if err != nil {
			return &p.Void, err
		}
	}

	if isProc {
		ex, err = proc.Fn(&args)
	} else if isLambda {
		paramLen := len(lambda.Params.Lst)
		argsLen := len(args.Lst)
		if paramLen != argsLen {
			return &p.Void, newError(errArityMismatch, lambda.Name, strconv.Itoa(paramLen), strconv.Itoa(argsLen))
		}

		lambdaEnv := makeEnvironment(env, lambda.Params, &args)
		for _, expr := range lambda.Body.Lst {
			ex, err = lambdaEnv.eval(expr)
			if err != nil {
				break
			}
		}
	}

	return ex, err
}

// (define <identifier> <expression>)
// or
// (define (<lambda name> [args...]) <lambda body expressions...>)
func (env *environment) evalDefine(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(lst.Lst)
	if len < 3 {
		return &p.Void, newError(errBadSyntax, "define", "at least 2 arguments", strconv.Itoa(len-1))
	}

	if len > 3 {
		if _, isLst := lst.Lst[1].(*p.ExprList); !isLst {
			return &p.Void, newError(errBadSyntax, "define", "exactly one expression after identifier")
		}
	}

	var ident string

	switch firstArg := lst.Lst[1].(type) {
	case *p.ExprList: // Lambda definition
		if lambdaName, isVar := firstArg.Lst[0].(*p.Variable); isVar {
			ident = lambdaName.Val
			params := p.ExprList{Lst: firstArg.Lst[1:]}
			body := p.ExprList{Lst: lst.Lst[2:]}
			ex = &p.Lambda{Name: ident, Params: &params, Body: &body}
		} else {
			return &p.Void, newError(errBadSyntax, "define", "identifier", firstArg.Lst[0].String(0))
		}

	case *p.Variable: // Variable definition
		ident = firstArg.Val
		ex, err = env.eval(lst.Lst[2])
		if err != nil {
			return &p.Void, err
		}

	default:
		return &p.Void, newError(errBadSyntax, "define", "identifier or list", lst.Lst[1].String(0))
	}

	env.vars[ident] = ex

	return ex, nil
}

// (cond (<clause condition> <clause result>) ... [(else <clause result>)])
func (env *environment) evalCond(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	for _, ex := range lst.Lst[1:len(lst.Lst)] {
		clause, isPair := isPair(ex)
		if !isPair {
			return &p.Void, newError(errBadSyntax, "cond", "pair? as a test clause", ex.String(0))
		}

		testClause := clause.Lst[0]
		resClauses := clause.Lst[1:len(clause.Lst)]
		isClauseTrue := false

		if varTest, isVar := testClause.(*p.Variable); isVar {
			if varTest.Val == "else" {
				isClauseTrue = true
			}
		}

		if !isClauseTrue {
			clRes, err := env.eval(testClause)
			if err != nil {
				return &p.Void, err
			}

			if !p.IsFalseSym(clRes) {
				isClauseTrue = true
			}
		}

		if isClauseTrue {
			var res p.Expression = &p.Void
			for _, ex := range resClauses {
				res, err = env.eval(ex)
				if err != nil {
					return &p.Void, err
				}
			}
			return res, nil
		}
	}

	return &p.Void, nil
}

// (lambda (<parameters...>) <body expressions>)
func (env *environment) evalLambda(lst *p.ExprList) (ex p.Expression, err *p.Error) {
	lstLen := len(lst.Lst)
	if lstLen < 3 {
		return &p.Void, newError(errBadSyntax, "lambda", "at least 2 arguments", strconv.Itoa(lstLen-1))
	}

	params, isLst := lst.Lst[1].(*p.ExprList)
	if !isLst {
		return &p.Void, newError(errBadSyntax, "lambda", "a list of parameters", lst.Lst[0].String(0))
	}

	res := &p.Lambda{Params: params, Body: &p.ExprList{}}
	res.Body.Lst = lst.Lst[2:lstLen]

	return res, nil
}

/// ------------------------------------------------------------------------ ///
/// ---------------------- Default procedure methods ----------------------- ///
/// ------------------------------------------------------------------------ ///

// (- [numbers...])
func procSubtract(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procSubDiv(args, true)
}

// (/ [numbers...])
func procDivide(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procSubDiv(args, false)
}

// (+ [numbers...])
func procAdd(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procAddMult(args, true)
}

// (+ [numbers...])
func procMultiply(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procAddMult(args, false)
}

// (< [numbers...])
func procLess(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procComp(args, less)
}

// (<= [numbers...])
func procLessEq(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procComp(args, lessEq)
}

// (> [numbers...])
func procGreater(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procComp(args, greater)
}

// (>= [numbers...])
func procGreaterEq(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procComp(args, greaterEq)
}

// (= [numbers...])
func procEquals(args *p.ExprList) (ex p.Expression, err *p.Error) {
	return procComp(args, equal)
}

// (number? <number>)
func procIsNumber(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 1 {
		return &p.Void, newError(errArityMismatch, "number?", "1", strconv.Itoa(len))
	}

	if _, isNum := args.Lst[0].(*p.Number); isNum {
		return &p.TrueSym, nil
	}

	return &p.FalseSym, nil
}

// (null? <expression>)
func procIsNull(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 1 {
		return &p.Void, newError(errArityMismatch, "null?", "1", strconv.Itoa(len))
	}

	if p.IsNullSym(args.Lst[0]) {
		return &p.TrueSym, nil
	}

	return &p.FalseSym, nil
}

// (and [args...])
func procAnd(args *p.ExprList) (ex p.Expression, err *p.Error) {
	res := p.Expression(&p.TrueSym)

	for _, ex := range args.Lst {
		if p.IsFalseSym(ex) {
			return &p.FalseSym, nil
		}

		res = ex
	}

	return res, nil
}

// (or [args...])
func procOr(args *p.ExprList) (ex p.Expression, err *p.Error) {
	for _, ex := range args.Lst {
		if !p.IsFalseSym(ex) {
			return ex, nil
		}
	}

	return &p.FalseSym, nil
}

// (remainder <dividend> <divisor>)
func procRemainder(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 2 {
		return &p.Void, newError(errArityMismatch, "remainder", "2", strconv.Itoa(len))
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		return &p.Void, newError(errContractViolation, "remainder", "number?", args.Lst[0].String(0))
	}

	div, isNumDiv := args.Lst[1].(*p.Number)
	if !isNumDiv {
		return &p.Void, newError(errContractViolation, "remainder", "number?", args.Lst[1].String(0))
	}

	return &p.Number{Val: float64(int64(num.Val) % int64(div.Val))}, nil
}

// (quotient <dividend> <divisor>)
func procQuotient(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 2 {
		return &p.Void, newError(errArityMismatch, "quotient", "2", strconv.Itoa(len))
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		return &p.Void, newError(errContractViolation, "quotient", "number?", args.Lst[0].String(0))
	}

	div, isNumDiv := args.Lst[1].(*p.Number)
	if !isNumDiv {
		return &p.Void, newError(errContractViolation, "quotient", "number?", args.Lst[1].String(0))
	}

	return &p.Number{Val: float64(int64(num.Val) / int64(div.Val))}, nil
}

// (expt <base> <exponent>)
func procExpt(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 2 {
		return &p.Void, newError(errArityMismatch, "expt", "2", strconv.Itoa(len))
	}

	num, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		return &p.Void, newError(errContractViolation, "expt", "number?", args.Lst[0].String(0))
	}

	exp, isExpDiv := args.Lst[1].(*p.Number)
	if !isExpDiv {
		return &p.Void, newError(errContractViolation, "expt", "number?", args.Lst[1].String(0))
	}

	return &p.Number{Val: math.Pow(num.Val, exp.Val)}, nil
}

// (list [args...])
func procList(args *p.ExprList) (ex p.Expression, err *p.Error) {
	if len(args.Lst) == 0 {
		return &p.NullSym, nil
	}

	args.Lst = append(args.Lst, &p.NullSym)

	return args, nil
}

// (cons <first> <second>)
func procCons(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 2 {
		return &p.Void, newError(errArityMismatch, "cons", "2", strconv.Itoa(len))
	}

	resLst := args.Lst[0:2]
	if secArg, isLst := args.Lst[1].(*p.ExprList); isLst && secArg.Qlevel <= 1 {
		resLst = append(args.Lst[0:1], secArg.Lst...)
	}

	return &p.ExprList{Lst: resLst, Qlevel: 1}, nil
}

// (car <pair>)
func procCar(args *p.ExprList) (ex p.Expression, err *p.Error) {
	len := len(args.Lst)
	if len != 1 {
		return &p.Void, newError(errArityMismatch, "car", "1", strconv.Itoa(len))
	}

	arg := args.Lst[0]
	if lstArg, isLst := arg.(*p.ExprList); isLst {
		return lstArg.Lst[0], nil
	}

	return &p.Void, newError(errContractViolation, "car", "pair?", arg.String(0))
}

// (cdr <pair>)
func procCdr(args *p.ExprList) (ex p.Expression, err *p.Error) {
	argsLen := len(args.Lst)
	if argsLen != 1 {
		return &p.Void, newError(errArityMismatch, "cdr", "1", strconv.Itoa(argsLen))
	}

	arg := args.Lst[0]
	if pairArg, isPair := isPair(arg); isPair {
		if len(pairArg.Lst) == 2 {
			return pairArg.Lst[1], nil
		}

		return &p.ExprList{Lst: pairArg.Lst[1:], Qlevel: pairArg.Qlevel}, nil
	}

	return &p.Void, newError(errContractViolation, "cdr", "pair?", arg.String(0))
}

// (list? <expression>)
func procIsList(args *p.ExprList) (ex p.Expression, err *p.Error) {
	argsLen := len(args.Lst)
	if argsLen != 1 {
		return &p.Void, newError(errArityMismatch, "list?", "1", strconv.Itoa(argsLen))
	}

	arg := args.Lst[0]
	if p.IsNullSym(arg) {
		return &p.TrueSym, nil
	}

	if lst, isList := arg.(*p.ExprList); isList {
		len := len(lst.Lst)
		if len == 0 || p.IsNullSym(lst.Lst[len-1]) {
			return &p.TrueSym, nil
		}
	}

	return &p.FalseSym, nil
}

// (pair? <expression>)
func procIsPair(args *p.ExprList) (ex p.Expression, err *p.Error) {
	argsLen := len(args.Lst)
	if argsLen != 1 {
		return &p.Void, newError(errArityMismatch, "pair?", "1", strconv.Itoa(argsLen))
	}

	if _, isPair := isPair(args.Lst[0]); isPair {
		return &p.TrueSym, nil
	}

	return &p.FalseSym, nil
}

// (max <numbers...>)
func procMax(args *p.ExprList) (ex p.Expression, err *p.Error) {
	_, max, err := minMax(args)
	if err != nil {
		return &p.Void, err
	}

	return max, nil
}

// (min <numbers...>)
func procMin(args *p.ExprList) (ex p.Expression, err *p.Error) {
	min, _, err := minMax(args)
	if err != nil {
		return &p.Void, err
	}

	return min, nil
}

/// ------------------------------------------------------------------------ ///
/// --------------------------- Helper functions --------------------------- ///
/// ------------------------------------------------------------------------ ///

// (- [numbers...]) or (/ [numbers...])
func procSubDiv(args *p.ExprList, isSub bool) (ex p.Expression, err *p.Error) {
	res := 0.0
	if !isSub {
		res = 1
	}

	if len(args.Lst) == 0 {
		return &p.Number{Val: res}, nil
	}

	procName := "/"
	if isSub {
		procName = "-"
	}

	fnum, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		return &p.Void, newError(errContractViolation, procName, "number?", args.Lst[0].String(0))
	}
	res = fnum.Val

	if len(args.Lst) == 1 {
		if isSub {
			return &p.Number{Val: -res}, nil
		} else {
			return &p.Number{Val: 1 / res}, nil
		}
	}

	for _, ex := range args.Lst[1:] {
		num, isNum := ex.(*p.Number)
		if !isNum {
			return &p.Void, newError(errContractViolation, procName, "number?", args.Lst[0].String(0))
		}
		if isSub {
			res -= num.Val
		} else {
			res /= num.Val
		}
	}

	return &p.Number{Val: res}, nil
}

// (+ [numbers...]) or (* [numbers...])
func procAddMult(args *p.ExprList, isAdd bool) (ex p.Expression, err *p.Error) {
	res := 0.0
	if !isAdd {
		res = 1
	}

	procName := "*"
	if isAdd {
		procName = "+"
	}

	for _, ex := range args.Lst {
		num, isNum := ex.(*p.Number)
		if !isNum {
			return &p.Void, newError(errContractViolation, procName, "number?", ex.String(0))
		}

		if isAdd {
			res += num.Val
		} else {
			res *= num.Val
		}
	}

	return &p.Number{Val: res}, nil
}

// (<comparison character> [args...])
func procComp(args *p.ExprList, comp func(*p.Number, *p.Number) bool) (ex p.Expression, err *p.Error) {
	if len(args.Lst) == 0 {
		return &p.TrueSym, nil
	}

	lastNum, isNum := args.Lst[0].(*p.Number)
	if !isNum {
		return &p.Void, newError(errContractViolation, "<comparison>", "number?", args.Lst[0].String(0))
	}

	for _, ex := range args.Lst[1:] {
		num, isNum := ex.(*p.Number)
		if !isNum {
			return &p.Void, newError(errContractViolation, "<comparison>", "number?", ex.String(0))
		}

		if !comp(lastNum, num) {
			return &p.FalseSym, nil
		}

		lastNum = num
	}

	return &p.TrueSym, nil
}

// returns true if lhs < rhs
func less(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val < rhs.Val
}

// returns true if lhs <= rhs
func lessEq(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val <= rhs.Val
}

// returns true if lhs > rhs
func greater(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val > rhs.Val
}

// returns true if lhs >= rhs
func greaterEq(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val >= rhs.Val
}

// returns true if lhs == rhs
func equal(lhs *p.Number, rhs *p.Number) bool {
	return lhs.Val == rhs.Val
}

// test whether the given expression is a scheme pair
func isPair(arg p.Expression) (pair *p.ExprList, isPair bool) {
	if pair, isPair := arg.(*p.ExprList); isPair && len(pair.Lst) >= 2 {
		return pair, isPair
	}

	return nil, false
}

// returns the min and max number from the given list or error
func minMax(args *p.ExprList) (min *p.Number, max *p.Number, err *p.Error) {
	argsLen := len(args.Lst)
	if argsLen == 0 {
		return nil, nil, newError(errArityMismatch, "min/max", "1", strconv.Itoa(argsLen))
	}

	max, isNum := args.Lst[0].(*p.Number)
	min, _ = args.Lst[0].(*p.Number)
	if !isNum {
		return nil, nil, newError(errContractViolation, "min/max", "number?", args.Lst[0].String(0))
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
			return nil, nil, newError(errContractViolation, "min/max", "number?", expr.String(0))
		}
	}

	return min, max, nil
}
