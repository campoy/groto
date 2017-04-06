package scanner

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/campoy/groto/token"
)

func New(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

type Scanner struct {
	r *bufio.Reader
}

type Token struct {
	Kind token.Kind
	Text string
}

func (t Token) String() string {
	var name string
	for _, r := range t.Kind.String() {
		if unicode.IsUpper(r) {
			name += fmt.Sprintf(" %c", unicode.ToLower(r))
		} else {
			name += string(r)
		}
	}
	name = strings.TrimSpace(name)
	if t.Text == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, t.Text)
}

func (s *Scanner) emit(kind token.Kind, value []rune) Token {
	return Token{Kind: kind, Text: string(value)}
}

func (s *Scanner) Scan() (tok Token) {
	s.readWhile(isSpace)

	r := s.peek()
	switch {
	case r == eof:
		return s.emit(token.EOF, nil)
	case isLetter(r):
		return s.identifier()
	case isDecimalDigit(r):
		return s.number()
	case r == quote || r == doubleQuote:
		return s.string()
	case r == '/':
		return s.comment()
	case token.Punctuation(string(r)) != token.Illegal:
		s.read()
		return s.emit(token.Punctuation(string(r)), nil)
	default:
		s.read()
		return s.emit(token.Illegal, []rune{r})
	}
}

// IDENTIFIERS

func (s *Scanner) identifier() Token {
	value := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))

	switch text := string(value); {
	case text == "true":
		return s.emit(token.True, nil)
	case text == "false":
		return s.emit(token.False, nil)
	case token.Keyword(text) != token.Illegal:
		return s.emit(token.Keyword(text), nil)
	case token.Type(text) != token.Illegal:
		return s.emit(token.Type(text), nil)
	default:
		return s.emit(token.Identifier, value)
	}
}

// STRINGS

func (s *Scanner) string() Token {
	first := s.read()
	value := []rune{first}
	for {
		value = append(value, s.readUntil(equals(first))...)
		value = append(value, s.read())
		if len(value) == 2 || value[len(value)-2] != backslash {
			return s.emit(token.StringLiteral, value)
		}
	}
}

// COMMENTS

func (s *Scanner) comment() Token {
	value := []rune{s.read(), s.read()}
	if string(value) != "//" {
		return s.emit(token.Illegal, value)
	}

	value = append(value, s.readUntil(equals('\n'))...)
	return s.emit(token.Comment, value)
}

// NUMBERS

func (s *Scanner) number() Token {
	first := s.read()
	second := s.peek()

	if first == '0' && isDecimalDigit(second) {
		return s.octal([]rune{first})
	}
	if first == '0' && (second == 'x' || second == 'X') {
		s.read()
		return s.hex([]rune{first, second})
	}

	tok := token.DecimalLiteral
	value := []rune{first}
	value = append(value, s.readWhile(isDecimalDigit)...)

	next := s.peek()
	if next == dot {
		s.read()
		tok = token.FloatLiteral
		value = append(value, dot)
		value = append(value, s.readWhile(isDecimalDigit)...)
		next = s.peek()
	}

	if next == 'E' || next == 'e' {
		s.read()
		tok = token.FloatLiteral
		value = append(value, next)

		sign := s.read()
		value = append(value, sign)
		if sign != '+' && sign != '-' {
			return s.emit(token.Illegal, value)
		}
		value = append(value, s.readWhile(isDecimalDigit)...)
	}

	return s.emit(tok, value)
}

func (s *Scanner) octal(value []rune) Token {
	value = append(value, s.readWhile(isOctalDigit)...)
	if isDecimalDigit(s.peek()) {
		return s.emit(token.Illegal, append(value, s.read()))
	}
	return s.emit(token.OctalLiteral, value)
}

func (s *Scanner) hex(value []rune) Token {
	value = append(value, s.readWhile(isHexDigit)...)
	if len(value) == 2 {
		return s.emit(token.Illegal, value)
	}
	return s.emit(token.HexLiteral, value)
}

// Utility functions for the scanner

func (s *Scanner) read() rune {
	r, _, err := s.r.ReadRune()
	if err == io.EOF {
		return eof
	}
	return r
}

func (s *Scanner) unread() {
	if err := s.r.UnreadRune(); err != nil {
		panic(err)
	}
}

func (s *Scanner) peek() rune {
	r := s.read()
	if r != eof {
		s.unread()
	}
	return r
}

func (s *Scanner) readUntil(p runePredicate) []rune {
	var value []rune
	for {
		r := s.read()
		if r == eof {
			return value
		}
		if p(r) {
			s.unread()
			return value
		}
		value = append(value, r)
	}
}

func (s *Scanner) readWhile(p runePredicate) []rune { return s.readUntil(not(p)) }

// Character classes

type runePredicate func(rune) bool

var (
	isLetter       = unicode.IsLetter
	isSpace        = unicode.IsSpace
	isDecimalDigit = isBetween('0', '9')
	isOctalDigit   = isBetween('0', '7')
	isHexDigit     = or(isDecimalDigit, isBetween('a', 'f'), isBetween('A', 'F'))
)

func isBetween(a, b rune) runePredicate { return func(r rune) bool { return r >= a && r <= b } }
func equals(r rune) runePredicate       { return func(s rune) bool { return r == s } }
func not(f runePredicate) runePredicate { return func(r rune) bool { return !f(r) } }

func or(fs ...runePredicate) runePredicate {
	return func(r rune) bool {
		for _, f := range fs {
			if f(r) {
				return true
			}
		}
		return false
	}
}

const (
	lineBreak   rune = '\n'
	underscore  rune = '_'
	eof         rune = 0
	dot         rune = '.'
	backslash   rune = '\\'
	quote       rune = '\''
	doubleQuote rune = '"'
)
