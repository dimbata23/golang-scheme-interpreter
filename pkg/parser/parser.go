package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/lexer"
)

type Expression interface {
	String(qlevel int) string
}

type Error struct {
	Val string
}

func (e *Error) String() string {
	return e.Val
}

type Number struct {
	Val    float64
	qlevel int
}

func getQs(exQLevel int, currQLevel int) string {
	qs := ""
	if exQLevel > currQLevel {
		qs = strings.Repeat("'", exQLevel-currQLevel)
	}

	return qs
}

func (n *Number) String(qlevel int) string {
	return getQs(n.qlevel, qlevel+1) + strconv.FormatFloat(n.Val, 'f', -1, 64)
}

type Variable struct {
	Val string
}

func (v *Variable) String(_ int) string {
	return v.Val
}

type ExprList struct {
	Lst    []interface{ Expression }
	Qlevel int
}

func (l *ExprList) String(qlevel int) string {
	res := getQs(l.Qlevel, qlevel)

	len := len(l.Lst)
	if len == 0 {
		return res + "()"
	}

	res += "("

	for i, expr := range l.Lst[0 : len-1] {
		if i != 0 {
			res += " "
		}
		res += expr.String(l.Qlevel + 1)
	}

	lastExpr := l.Lst[len-1]
	if !IsNullSym(lastExpr) {
		res += " . " + lastExpr.String(l.Qlevel+1)
	}

	return res + ")"
}

type Procedure struct {
	Fn func(*ExprList) (Expression, *Error)
}

func (proc *Procedure) String(_ int) string {
	return "#<procedure>"
}

type Lambda struct {
	Name   string
	Params *ExprList
	Body   *ExprList
}

func (lambda *Lambda) String(_ int) string {
	if len(lambda.Name) != 0 {
		return fmt.Sprintf("#<lambda %s>", lambda.Name)
	}

	return "#<lambda>"
}

type Symbol struct {
	val    string
	qlevel int
}

var NullSym = Symbol{val: "()", qlevel: 1}
var FalseSym = Symbol{val: "#f", qlevel: 1}
var TrueSym = Symbol{val: "#t", qlevel: 1}

func IsNullSym(expr Expression) bool {
	if s, isSym := expr.(*Symbol); isSym {
		return *s == NullSym
	}

	return false
}

func IsFalseSym(expr Expression) bool {
	if s, isSym := expr.(*Symbol); isSym {
		return *s == FalseSym
	}

	return false
}

func (s *Symbol) String(qlevel int) string {
	qs := s.qlevel
	if len(s.val) > 0 && s.val[0] == '#' {
		qs -= 1
	}
	return getQs(qs, qlevel) + s.val
}

type SpecialType int

const (
	SpecialExit SpecialType = iota
	SpecialCloseBracket
)

type SpecialExpr struct {
	typ SpecialType
}

func (s *SpecialExpr) String(_ int) string {
	switch s.typ {
	case SpecialExit:
		return "#<exit>"
	case SpecialCloseBracket:
		return "Unexpected `)`"
	}

	return "Unknown special expression"
}

var VoidExpr voidExpr = voidExpr{}

type voidExpr struct{}

func (ve *voidExpr) String(_ int) string {
	return "#<void>"
}

func IsSpecialExit(e Expression) bool {
	s, isSpec := e.(*SpecialExpr)
	return isSpec && s.typ == SpecialExit
}

type Parser struct {
	lexer *lexer.Lexer
}

func Parse(input string) *Parser {
	return &Parser{
		lexer: lexer.Lex(input),
	}
}

func (p *Parser) Next() (ex Expression, err *Error) {
	return p.next(0)
}

func (p *Parser) next(qlevel int) (ex Expression, err *Error) {
	token := p.lexer.NextToken()
	if token == nil {
		return nil, nil
	}

	switch token.Typ {

	case lexer.TokenError:
		return &VoidExpr, &Error{Val: token.Val}

	case lexer.TokenEOF:
		return nil, nil

	case lexer.TokenNumber:
		num, err := strconv.ParseFloat(token.Val, 64)
		if err != nil {
			return &VoidExpr, &Error{Val: err.Error()}
		}
		return &Number{Val: num, qlevel: qlevel}, nil

	case lexer.TokenIdentifier:
		if qlevel == 0 {
			return &Variable{Val: token.Val}, nil
		}

		return &Symbol{val: token.Val, qlevel: qlevel}, nil

	case lexer.TokenString:
		panic("not implemented")

	case lexer.TokenOpenBracket:
		res := ExprList{Lst: make([]interface{ Expression }, 0), Qlevel: qlevel}

		for {
			inexpr, err := p.next(qlevel)
			if err != nil {
				return &VoidExpr, err
			}

			s, isSpec := inexpr.(*SpecialExpr)
			if isSpec && s.typ == SpecialCloseBracket {
				break
			}

			if inexpr == nil {
				return &VoidExpr, &Error{Val: "read-syntax: expected a `)` to close `(`"}
			}

			res.Lst = append(res.Lst, inexpr)
		}

		if len(res.Lst) == 0 && res.Qlevel == 1 {
			return &NullSym, nil
		}

		if len(res.Lst) == 1 && res.Qlevel == 0 {
			s, isSpec := res.Lst[0].(*Variable)
			if isSpec && s.Val == "exit" {
				return &SpecialExpr{typ: SpecialExit}, nil
			}
		}

		if res.Qlevel > 0 {
			res.Lst = append(res.Lst, &NullSym)
		}

		return &res, nil

	case lexer.TokenCloseBracket:
		return &SpecialExpr{typ: SpecialCloseBracket}, nil

	case lexer.TokenQuote:
		return p.next(qlevel + 1)

	case lexer.TokenSkip:
		panic("supposedly unreachable code?")

	case lexer.TokenOutsideBrackets:
		return p.next(qlevel) // skip anything outside brackets
	}

	return &VoidExpr, &Error{Val: "read-syntax: unknown lex type"}
}
