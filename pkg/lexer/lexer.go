// A very simple lexer inspired by Lexical Scanning in Go, a talk by Rob Pike
// at Google Technology User Group given on Tuesday, 30 August 2011.

package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const eof rune = -1

const (
	TokenError TokenType = iota // lexer error occured; value is the error message
	TokenEOF
	TokenNumber
	TokenIdentifier
	TokenString
	TokenOpenBracket
	TokenCloseBracket
	TokenQuote
	TokenSkip
	TokenOutsideBrackets
)

type Token struct {
	Typ TokenType
	Val string
}

// Debug info
func (i Token) String() string {
	switch i.Typ {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "Err: " + i.Val
	}

	str := fmt.Sprintf("(%q, ", i.Val)
	switch i.Typ {
	case TokenNumber:
		str += "Number"
	case TokenIdentifier:
		str += "Identifier"
	case TokenString:
		str += "String"
	case TokenOpenBracket:
		str += "OpenBracket"
	case TokenCloseBracket:
		str += "CloseBracket"
	case TokenQuote:
		str += "Quote"
	case TokenSkip:
		str += "Skip"
	case TokenOutsideBrackets:
		str += "OutsideBrackets"
	}

	return str + ")"
}

// Lexer struct

type Lexer struct {
	input  string     // text being lexed
	start  int        // starting position of current token
	pos    int        // current position in the text
	width  int        // width of last read rune
	state  stateFn    // the state function used for lexing
	level  int        // number of lists opened and not closed
	tokens chan Token // output channel of read token
}

type stateFn func(*Lexer) stateFn

func (l *Lexer) emit(t TokenType) {
	l.tokens <- Token{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *Lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width

	return r
}

func (l *Lexer) ignore() {
	l.start = l.pos
}

func (l *Lexer) backup() {
	l.pos -= l.width
}

func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *Lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- Token{TokenError, fmt.Sprintf(format, args...)}
	return nil
}

// consumes the next rune if it's from the valid set
func (l *Lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}

	l.backup()
	return false
}

// consumes multiple runes from the valid set
func (l *Lexer) acceptRun(valid string) int {
	cnt := 0
	for strings.ContainsRune(valid, l.next()) {
		cnt++
	}
	l.backup()
	return cnt
}

func Lex(input string) *Lexer {
	l := &Lexer{
		input:  input,
		state:  lexGeneral,
		tokens: make(chan Token, 2),
	}

	return l
}

func (l *Lexer) NextToken() *Token {
	for {
		select {
		case token := <-l.tokens:
			if token.Typ == TokenEOF {
				return nil
			}

			return &token
		default:
			if l.state != nil {
				l.state = l.state(l)
			} else {
				l.state = lexGeneral
			}
		}
	}
}

/// State functions

// func lexOutsideList(l *Lexer) stateFn {
// 	for {
// 		if strings.HasPrefix(l.input[l.pos:], "(") {
// 			if l.pos > l.start {
// 				l.emit(TokenOutsideBrackets)
// 			}
// 			return lexOpenBracket
// 		}
// 		if l.next() == eof {
// 			break
// 		}
// 	}

// 	if l.pos > l.start {
// 		l.emit(TokenOutsideBrackets)
// 	}

// 	l.emit(TokenEOF)

// 	return nil
// }

func lexOpenBracket(l *Lexer) stateFn {
	l.pos++
	l.level++
	l.emit(TokenOpenBracket)
	return lexGeneral
}

func lexGeneral(l *Lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], ")") {
			return lexCloseBracket
		}

		if strings.HasPrefix(l.input[l.pos:], "(") {
			return lexOpenBracket
		}

		switch r := l.next(); {
		case r == eof:
			if l.level > 0 {
				l.emit(TokenEOF)
				return l.errorf("unexpected end of file: expected a `)` to close `(`")
			} else {
				l.emit(TokenEOF)
				return lexGeneral
			}
		case unicode.IsSpace(r):
			l.ignore()
		case r == '"':
			return lexDoubleQuote
		case r == '\'':
			l.backup()
			return lexQuote
		case r == '+' || r == '-' || r == '.' || r >= '0' && r <= '9':
			l.backup()
			return lexNumber
		//case unicode.IsLetter(r):

		// TODO:
		default:
			//l.backup()
			return lexIdentifier
		}
	}
}

func lexNumber(l *Lexer) stateFn {
	// optional leading sign
	bSigned := l.accept("+-")
	const digits = "0123456789"
	cnt := l.acceptRun(digits)
	if l.accept(".") {
		cnt++
		cnt += l.acceptRun(digits)
	}

	extAlpha := "+-.*/<=>!?:$%_&~^"

	r := l.peek()

	if unicode.IsLetter(r) || strings.ContainsRune(extAlpha, r) {
		return lexIdentifier
	}

	if bSigned && cnt == 0 {
		// special case: just + or just -
		l.emit(TokenIdentifier)
		return lexGeneral
	}

	l.emit(TokenNumber)
	return lexGeneral
}

func lexCloseBracket(l *Lexer) stateFn {
	l.next()
	l.emit(TokenCloseBracket)
	if l.level > 0 {
		l.level--
	} else {
		return l.errorf("read-syntax: unexpected `)`")
	}

	if l.level > 0 {
		return lexGeneral
	}

	return lexGeneral
}

func lexDoubleQuote(l *Lexer) stateFn {
	for {
		r := l.next()
		if r == '"' {
			l.emit(TokenString)
			return lexGeneral
		} else if r == eof {
			return l.errorf("unexpected end of file: expected `\"` to close `\"`")
		}
	}
}

func lexQuote(l *Lexer) stateFn {
	l.pos++
	l.emit(TokenQuote)
	return lexGeneral
}

func lexIdentifier(l *Lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unexpected end of file: expected `)` to close `(`")

		// TODO: change
		case unicode.IsSpace(r) || r == ')':
			l.backup()
			l.emit(TokenIdentifier)
			return lexGeneral
		}
	}
}
