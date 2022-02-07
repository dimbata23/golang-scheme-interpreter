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
	Val    float64
	qlevel int
}

func (n *Number) String() string {
	return fmt.Sprintf("%f", n.Val)
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
	res := ""
	if l.Qlevel > 0 {
		res = strings.Repeat("'", l.Qlevel-1)
	}

	len := len(l.Lst)
	if len == 0 {
		return res + "()"
	}

	res += "("

	for i, expr := range l.Lst[0 : len-1] {
		if i != 0 {
			res += " "
		}
		res += expr.String()
	}

	lastExpr := l.Lst[len-1]
	if !IsNullSym(lastExpr) {
		res += " . " + lastExpr.String()
	}

	return res + ")"
}

type Procedure struct {
	Fn func(*ExprList) Expression
}

func (proc *Procedure) String() string {
	return "#<procedure>"
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

func (s *Symbol) String() string {
	qs := s.qlevel
	if *s == FalseSym || *s == TrueSym {
		qs -= 1
	}
	return strings.Repeat("'", qs) + s.val
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
		return &Number{Val: num, qlevel: qlevel}

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
			return &NullSym
		}

		if len(res.Lst) == 1 && res.Qlevel == 0 {
			s, isSpec := res.Lst[0].(*Variable)
			if isSpec && s.Val == "exit" {
				return &SpecialExpr{typ: SpecialExit}
			}
		}

		if res.Qlevel > 0 {
			res.Lst = append(res.Lst, &NullSym)
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
