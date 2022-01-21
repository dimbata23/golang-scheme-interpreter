package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/lexer"
)

type Expression interface {
	String() string // TODO: Add support of qlevels
}

type Error struct {
	val string
}

func (e *Error) String() string {
	return e.val
}

type Number struct {
	val    float64
	qlevel int
}

func (n *Number) String() string {
	return fmt.Sprintf("%f", n.val)
}

type Variable struct {
	Val string
}

func (v *Variable) String() string {
	//panic("not implemented")
	return v.Val
}

type ExprList struct {
	Lst    []interface{ Expression }
	Qlevel int
}

func (l *ExprList) String() string {
	res := strings.Repeat("'", l.Qlevel) + "("
	for i, expr := range l.Lst {
		if i != 0 {
			res += " "
		}
		res += expr.String()
	}

	return res + ")"
}

type Procedure struct {
	Fn func(*ExprList) Expression
}

func (proc *Procedure) String() string {
	panic("not implemented")
}

type Lambda struct {
	Name   string
	Params *ExprList
	Body   *ExprList
}

func (lambda *Lambda) String() string {
	return fmt.Sprintf("#<lambda %s>", lambda.Name)
}

type Symbol struct {
	val    string
	qlevel int
}

var nullsym = Symbol{val: "()", qlevel: 1}

func IsNullSym(expr Expression) bool {
	if s, isSym := expr.(*Symbol); isSym {
		return *s == nullsym
	}

	return false
}

func (s *Symbol) String() string {
	return s.val
}

type SpecialType int

const (
	SpecialExit SpecialType = iota
	SpecialCloseBracket
)

type SpecialExpr struct {
	typ SpecialType
}

func (s *SpecialExpr) String() string {
	switch s.typ {
	case SpecialExit:
		return "(exit)"
	case SpecialCloseBracket:
		return "Unexpected `)`"
	}

	return "Unknown special expression"
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

func (p *Parser) Next() Expression {
	return p.next(0)
}

func (p *Parser) next(qlevel int) Expression {
	token := p.lexer.NextToken()
	if token == nil {
		return nil
	}

	switch token.Typ {

	case lexer.TokenError:
		return &Error{val: token.Val}

	case lexer.TokenEOF:
		return nil

	case lexer.TokenNumber:
		num, err := strconv.ParseFloat(token.Val, 64)
		if err != nil {
			return &Error{val: err.Error()}
		}
		return &Number{val: num, qlevel: qlevel}

	case lexer.TokenIdentifier:
		if qlevel == 0 {
			return &Variable{Val: token.Val}
		}

		return &Symbol{val: token.Val, qlevel: qlevel}

	case lexer.TokenString:
		panic("not implemented")

	case lexer.TokenOpenBracket:
		res := ExprList{Lst: make([]interface{ Expression }, 0), Qlevel: qlevel}

		for {
			inexpr := p.next(qlevel)

			e, isErr := inexpr.(*Error)
			if isErr {
				return e
			}

			s, isSpec := inexpr.(*SpecialExpr)
			if isSpec && s.typ == SpecialCloseBracket {
				break
			}

			if inexpr == nil {
				return &Error{val: "Unexpected end of file: expected a `)` to close `(`"}
			}

			res.Lst = append(res.Lst, inexpr)
		}

		if len(res.Lst) == 0 && res.Qlevel == 1 {
			return &nullsym
		}

		if len(res.Lst) == 1 && res.Qlevel == 0 {
			s, isSpec := res.Lst[0].(*Variable)
			if isSpec && s.Val == "exit" {
				return &SpecialExpr{typ: SpecialExit}
			}
		}

		return &res

	case lexer.TokenCloseBracket:
		return &SpecialExpr{typ: SpecialCloseBracket}

	case lexer.TokenQuote:
		return p.next(qlevel + 1)

	case lexer.TokenSkip:
		panic("Supposedly unreachable code?")

	case lexer.TokenOutsideBrackets:
		return p.next(qlevel) // skip anything outside brackets
	}

	return &Error{val: "Unknown lexed type"}
}
