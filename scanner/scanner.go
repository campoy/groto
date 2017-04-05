package scanner

import (
	"bufio"
	"io"
	"unicode"

	"github.com/campoy/groto/token"
)

func New(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

type Scanner struct {
	r *bufio.Reader
}

func (s *Scanner) Scan() (token.Token, []rune) {
	s.readWhile(isSpace)

	r := s.peek()
	switch {
	case r == eof:
		return token.EOF, nil
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
		return token.Punctuation(string(r)), nil
	default:
		s.read()
		return token.Illegal, []rune{r}
	}
}

// IDENTIFIERS

func (s *Scanner) identifier() (token.Token, []rune) {
	value := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))
	if s.peek() == dot {
		return s.fullIdentifier(value)
	}

	switch text := string(value); {
	case text == "true":
		return token.True, nil
	case text == "false":
		return token.False, nil
	case token.Keyword(text) != token.Illegal:
		return token.Keyword(text), nil
	case token.Type(text) != token.Illegal:
		return token.Type(text), nil
	default:
		return token.Identifier, value
	}
}

func (s *Scanner) fullIdentifier(value []rune) (token.Token, []rune) {
	for s.peek() == dot {
		value = append(value, s.read())
		cont := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))
		value = append(value, cont...)
	}
	return token.FullIdentifier, value
}

// STRINGS

func (s *Scanner) string() (token.Token, []rune) {
	first := s.read()
	value := []rune{first}
	for {
		value = append(value, s.readUntil(equals(first))...)
		value = append(value, s.read())
		if len(value) == 2 || value[len(value)-2] != backslash {
			return token.StringLiteral, value
		}
	}
}

// COMMENTS

func (s *Scanner) comment() (token.Token, []rune) {
	value := []rune{s.read(), s.read()}
	if string(value) != "//" {
		return token.Illegal, value
	}

	value = append(value, s.readUntil(equals('\n'))...)
	return token.Comment, value
}

// NUMBERS

func (s *Scanner) number() (token.Token, []rune) {
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

	next := s.read()
	if next == dot {
		tok = token.FloatLiteral
		value = append(value, dot)
		value = append(value, s.readWhile(isDecimalDigit)...)
		next = s.read()
	}

	if next == 'E' || next == 'e' {
		tok = token.FloatLiteral
		value = append(value, next)

		sign := s.read()
		value = append(value, sign)
		if sign != '+' && sign != '-' {
			return token.Illegal, value
		}
		value = append(value, s.readWhile(isDecimalDigit)...)
	}

	return tok, value
}

func (s *Scanner) octal(value []rune) (token.Token, []rune) {
	value = append(value, s.readWhile(isOctalDigit)...)
	if isDecimalDigit(s.peek()) {
		return token.Illegal, append(value, s.read())
	}
	return token.OctalLiteral, value
}

func (s *Scanner) hex(value []rune) (token.Token, []rune) {
	value = append(value, s.readWhile(isHexDigit)...)
	if len(value) == 2 {
		return token.Illegal, value
	}
	return token.HexLiteral, value
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
