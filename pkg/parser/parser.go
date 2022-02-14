// A parser that uses the lexer package and creates
// a meaningful expression out of the lexed tokens
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dimbata23/golang-scheme-interpreter/pkg/lexer"
)

/// ------------------------------------------------------------------------ ///
/// -------------------------- Public definitions -------------------------- ///
/// ------------------------------------------------------------------------ ///

// the parser struct
type Parser struct {
	lexer *lexer.Lexer
}

// the basic expression interface
type Expression interface {
	String(qlevel int) string // returns string representation of the expression
}

// scheme number, can be a real or an integer
type Number struct {
	Val    float64
	qlevel int
}

// identifier (name) of a scheme variable
type Variable struct {
	Val string
}

// generic scheme list
type ExprList struct {
	Lst    []interface{ Expression }
	Qlevel int
}

// scheme procedure
type Procedure struct {
	Fn func(*ExprList) (Expression, *Error)
}

// scheme lambda function
type Lambda struct {
	Name   string    // name of the lambda (if given)
	Params *ExprList // list of parameter names
	Body   *ExprList // list of expressions inside the body
}

// scheme symbol
type Symbol struct {
	val    string
	qlevel int
}

// scheme void expression
type VoidExpr struct{}

var NullSym = Symbol{val: "()", qlevel: 1}  // the scheme null symbol
var FalseSym = Symbol{val: "#f", qlevel: 1} // the scheme false symbol
var TrueSym = Symbol{val: "#t", qlevel: 1}  // the scheme true symbol
var Void VoidExpr = VoidExpr{}              // the scheme void expression

// the error type used by the parser package
type Error struct {
	Val string // message about occured the error
}

// special type used for non-scheme related functionality of the parser
type SpecialType int

const (
	SpecialExit         SpecialType = iota // the (exit) command has been parsed
	SpecialCloseBracket                    // a lonely `)` has been parsed
)

// special expression used for non-scheme related functionality
type SpecialExpr struct {
	typ SpecialType
}

/// ------------------------------------------------------------------------ ///
/// --------------------- Public functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// creates a parser from the given input
func NewParser(input string) *Parser {
	return &Parser{
		lexer: lexer.NewLexer(input),
	}
}

// parses and returns the next expression (ex) or nil when the input has ended
// can return an error (err) containing information about what went wrong
func (p *Parser) Next() (ex Expression, err *Error) {
	return p.next(0)
}

// tests whether the given expression is the scheme null symbol
func IsNullSym(expr Expression) bool {
	if s, isSym := expr.(*Symbol); isSym {
		return *s == NullSym
	}

	return false
}

// tests whether the given expression is the scheme null symbol
// note: only #f is false, anything else is considered true in scheme
func IsFalseSym(expr Expression) bool {
	if s, isSym := expr.(*Symbol); isSym {
		return *s == FalseSym
	}

	return false
}

// tests whether the given expression is an (exit) command
func IsSpecialExit(expr Expression) bool {
	s, isSpec := expr.(*SpecialExpr)
	return isSpec && s.typ == SpecialExit
}

// returns the error message
func (e *Error) String() string {
	return e.Val
}

/// ------------------------------------------------------------------------ ///
/// -------------------- Private functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// the inner next(...) method
func (p *Parser) next(qlevel int) (ex Expression, err *Error) {
	token := p.lexer.NextToken()
	if token == nil {
		return nil, nil
	}

	switch token.Typ {

	case lexer.TokenError:
		return &Void, &Error{Val: token.Val}

	case lexer.TokenEOF:
		return nil, nil

	case lexer.TokenNumber:
		num, err := strconv.ParseFloat(token.Val, 64)
		if err != nil {
			return &Void, &Error{Val: err.Error()}
		}
		return &Number{Val: num, qlevel: qlevel}, nil

	case lexer.TokenIdentifier:
		if qlevel == 0 {
			return &Variable{Val: token.Val}, nil
		}

		return &Symbol{val: token.Val, qlevel: qlevel}, nil

	case lexer.TokenString:
		return &Symbol{val: token.Val, qlevel: qlevel}, nil

	case lexer.TokenOpenBracket:
		res := ExprList{Lst: make([]interface{ Expression }, 0), Qlevel: qlevel}

		for {
			inexpr, err := p.next(qlevel)
			if err != nil {
				return &Void, err
			}

			s, isSpec := inexpr.(*SpecialExpr)
			if isSpec && s.typ == SpecialCloseBracket {
				break
			}

			if inexpr == nil {
				return &Void, &Error{Val: "read-syntax: expected a `)` to close `(`"}
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
		return p.next(qlevel)
	}

	return &Void, &Error{Val: "read-syntax: unknown lex type"}
}

/// ------------------------------------------------------------------------ ///
/// --------------------------- String() methods --------------------------- ///
/// ------------------------------------------------------------------------ ///

func (n *Number) String(qlevel int) string {
	return getQs(n.qlevel, qlevel+1) + strconv.FormatFloat(n.Val, 'f', -1, 64)
}

func (v *Variable) String(_ int) string {
	return v.Val
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

func (proc *Procedure) String(_ int) string {
	return "#<procedure>"
}

func (lambda *Lambda) String(_ int) string {
	if len(lambda.Name) != 0 {
		return fmt.Sprintf("#<lambda %s>", lambda.Name)
	}

	return "#<lambda>"
}

func (s *Symbol) String(qlevel int) string {
	qs := s.qlevel
	if len(s.val) > 0 && s.val[0] == '#' {
		qs -= 1
	}
	return getQs(qs, qlevel) + s.val
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

func (ve *VoidExpr) String(_ int) string {
	return "#<void>"
}

/// ------------------------------------------------------------------------ ///
/// -------------------------- Utility functions --------------------------- ///
/// ------------------------------------------------------------------------ ///

// returns the number of quotes needed to be printed
// depending on the current quote level of the print
// function and the quote level of the expression
func getQs(exQLevel int, currQLevel int) string {
	qs := ""
	if exQLevel > currQLevel {
		qs = strings.Repeat("'", exQLevel-currQLevel)
	}

	return qs
}
