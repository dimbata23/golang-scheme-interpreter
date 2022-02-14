// A very simple lexer inspired by Lexical Scanning in Go, a talk by Rob Pike
// at Google Technology User Group given on Tuesday, 30 August 2011.
package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

/// ------------------------------------------------------------------------ ///
/// -------------------------- Public definitions -------------------------- ///
/// ------------------------------------------------------------------------ ///

// the lexer struct
type Lexer struct {
	input  string     // text being lexed
	start  int        // starting position of current token
	pos    int        // current position in the text
	width  int        // width of last read rune
	state  stateFn    // the state function used for lexing
	level  int        // number of lists opened and not closed
	tokens chan Token // output channel of read tokens
}

// the basic token (unit) used by the lexer
type Token struct {
	Typ TokenType
	Val string
}

// token type used by the lexer
type TokenType int

const (
	TokenError        TokenType = iota // lexer error; value is the error msg
	TokenEOF                           // input ended
	TokenNumber                        // a number, integer or real
	TokenIdentifier                    // identifier (name) accepted by scheme
	TokenString                        // a seq of runes surrounded by `"`
	TokenOpenBracket                   // an opening bracket `(`
	TokenCloseBracket                  // a closing bracket `)`
	TokenQuote                         // a quote `'`
	TokenSkip                          // any whitespace or ignored lex tokens
)

/// ------------------------------------------------------------------------ ///
/// --------------------- Public functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// creates a lexer from the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		state:  lexGeneral,
		tokens: make(chan Token, 2),
	}

	return l
}

// returns the next token from the input
// or nil when the input has finished
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

/// ------------------------------------------------------------------------ ///
/// --------------------------- Inner structure ---------------------------- ///
/// ------------------------------------------------------------------------ ///

const eof rune = -1 // the end of file rune

// state function type returning another state function
// after lexing a part of the input
type stateFn func(*Lexer) stateFn

/// ------------------------------------------------------------------------ ///
/// -------------------- Private functions and methods --------------------- ///
/// ------------------------------------------------------------------------ ///

// sends tokens through the channel
func (l *Lexer) emit(t TokenType) {
	l.tokens <- Token{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// get the next rune in the input or eof
func (l *Lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width

	return r
}

// ignore the current token
func (l *Lexer) ignore() {
	l.start = l.pos
}

// returns back the last read rune (can only be used once after next())
func (l *Lexer) backup() {
	l.pos -= l.width
}

// peeks at what the next rune in the input is
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// emits a formatted error token to the channel
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

/// ------------------------------------------------------------------------ ///
/// --------------------------- State functions ---------------------------- ///
/// ------------------------------------------------------------------------ ///

// reads and emits open bracket
func lexOpenBracket(l *Lexer) stateFn {
	l.pos++
	l.level++
	l.emit(TokenOpenBracket)
	return lexGeneral
}

// lexes the next rune and returns a state function based on it
// or nil on error
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
				return l.errorf("expected a `)` to close `(`")
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

		default:
			return lexIdentifier
		}
	}
}

// reads and emits a number token
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

// reads and emits a closing bracket token
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

// reads and emits a string token
func lexDoubleQuote(l *Lexer) stateFn {
	for {
		r := l.next()
		if r == '"' {
			l.emit(TokenString)
			return lexGeneral
		} else if r == eof {
			return l.errorf("expected a `\"` to close `\"`")
		}
	}
}

// reads and emits a quote token
func lexQuote(l *Lexer) stateFn {
	l.pos++
	l.emit(TokenQuote)
	return lexGeneral
}

// reads and emits an identifier token
func lexIdentifier(l *Lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("expected a `)` to close `(`")

		case unicode.IsSpace(r) || r == ')':
			l.backup()
			l.emit(TokenIdentifier)
			return lexGeneral
		}
	}
}

/// ------------------------------------------------------------------------ ///
/// -------------------------- Utility functions --------------------------- ///
/// ------------------------------------------------------------------------ ///

// used for debug info
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
	}

	return str + ")"
}
