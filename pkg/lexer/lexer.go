package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const eof rune = -1

const (
	ItemError itemType = iota // lexer error occured; value is the error message
	ItemEOF
	ItemNumber
	ItemIdentifier
	ItemString
	ItemOpenBracket
	ItemCloseBracket
	ItemQuote
	ItemSkip
	ItemText
)

type Item struct {
	Typ itemType
	Val string
}

func (i Item) String() string {
	switch i.Typ {
	case ItemEOF:
		return "EOF"
	case ItemError:
		return i.Val
	}

	if len(i.Val) > 16 {
		return fmt.Sprintf("%.10q...", i.Val)
	}

	return fmt.Sprintf("%q", i.Val)
}

// Lexer struct

type lexer struct {
	input string    // text being lexed
	start int       // starting position of current item
	pos   int       // current position in the text
	width int       // width of last read rune
	state stateFn   // the state function used for lexing
	level int       // number of lists opened and not closed
	items chan Item // output channel of read items
}

type stateFn func(*lexer) stateFn

// func (l *lexer) run() {
// 	for state := lexText; state != nil; {
// 		state = state(l)
// 	}
// 	close(l.items)
// }

func (l *lexer) emit(t itemType) {
	l.items <- Item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width

	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- Item{ItemError, fmt.Sprintf(format, args...)}
	return nil
}

// consumes the next rune if it's from the valid set
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}

	l.backup()
	return false
}

// consumes multiple runes from the valid set
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func Lex(input string) *lexer {
	l := &lexer{
		input: input,
		state: lexText,
		items: make(chan Item, 2),
	}

	return l
}

func (l *lexer) NextItem() *Item {
	for {
		select {
		case item := <-l.items:
			if item.Typ == ItemEOF {
				return nil
			}

			return &item
		default:
			l.state = l.state(l)
		}
	}
}

/// State functions

func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], "(") {
			if l.pos > l.start {
				l.emit(ItemText)
			}
			return lexOpenBracket
		}
		if l.next() == eof {
			break
		}
	}

	if l.pos > l.start {
		l.emit(ItemText)
	}

	l.emit(ItemEOF)

	return nil
}

func lexOpenBracket(l *lexer) stateFn {
	l.pos++
	l.level++
	l.emit(ItemOpenBracket)
	return lexInsideList
}

func lexInsideList(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], ")") {
			return lexCloseBracket
		}

		if strings.HasPrefix(l.input[l.pos:], "(") {
			return lexOpenBracket
		}

		switch r := l.next(); {
		case r == eof:
			return l.errorf("unexpected end of file: expected `)` to close `(`")
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

func lexNumber(l *lexer) stateFn {
	// optional leading sign
	l.accept("+-")
	const digits = "0123456789"
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}

	extAlpha := "+-.*/<=>!?:$%_&~^"

	r := l.peek()

	if unicode.IsLetter(r) || strings.ContainsRune(extAlpha, r) {
		return lexIdentifier
	}

	l.emit(ItemNumber)
	return lexInsideList
}

func lexCloseBracket(l *lexer) stateFn {
	l.next()
	l.emit(ItemCloseBracket)
	if l.level > 0 {
		l.level--
	} else {
		return l.errorf("read-syntax: unexpected `)`")
	}

	if l.level > 0 {
		return lexInsideList
	}

	return lexText
}

func lexDoubleQuote(l *lexer) stateFn {
	for {
		r := l.next()
		if r == '"' {
			l.emit(ItemString)
			return lexInsideList
		} else if r == eof {
			return l.errorf("unexpected end of file: expected `\"` to close `\"`")
		}
	}
}

func lexQuote(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unexpected end of file: expected `)` to close `(`")
		case unicode.IsSpace(r) || r == ')':
			l.backup()
			l.emit(ItemQuote)
			return lexInsideList
		}
	}
}

func lexIdentifier(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unexpected end of file: expected `)` to close `(`")

		// TODO: change
		case unicode.IsSpace(r) || r == ')':
			l.backup()
			l.emit(ItemIdentifier)
			return lexInsideList
		}
	}
}
