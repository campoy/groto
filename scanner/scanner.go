package scanner

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
)

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

type Scanner struct {
	r *bufio.Reader
}

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

type Token interface {
	String() string
}

type eofToken struct{}

func (tok eofToken) String() string { return "EOF" }

var EOF Token = eofToken{}

type Error struct{ msg string }

func errorf(format string, args ...interface{}) Token { return Error{fmt.Sprintf(format, args...)} }
func (err Error) String() string                      { return err.msg }
func (err Error) Error() string                       { return err.msg }

func (s *Scanner) Scan() Token {
	s.readWhile(isSpace)

	r := s.peek()
	switch {
	case r == eof:
		return EOF
	case isLetter(r):
		return s.identifier()
	case isDecimalDigit(r):
		return s.number()
	case r == quote || r == doubleQuote:
		return s.string()
	default:
		return errorf("unexpected character %c", r)
	}
}

// IDENTIFIERS

type runes []rune

func (r runes) String() string { return string(r) }

type Identifier struct{ runes }

type Boolean struct{ value bool }

func (b Boolean) String() string {
	if b.value {
		return "true"
	}
	return "false"
}

// 	ident = letter { letter | decimalDigit | "_" }
func (s *Scanner) identifier() Token {
	value := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))

	switch s := string(value); s {
	case "true", "false":
		return Boolean{s == "true"}

	}
	// check if keyword
	// nan, inf

	return Identifier{value}
}

// STRINGS

type String struct{ runes }

func (s *Scanner) string() Token {
	first := s.read()
	var value []rune

	for {
		cont := s.readUntil(equals(first))
		next := s.read()
		value = append(value, cont...)

		if len(cont) == 0 || cont[len(cont)-1] != backslash {
			return String{value}
		}
		value = append(value, next)
	}
}

// NUMBERS

type Number interface {
	Token
	Float() float64
}
type Integer interface {
	Token
	Integer() int64
}

type SignedInteger struct {
	positive bool
	integer  Integer
}

func (s SignedInteger) String() string {
	sign := "+"
	if !s.positive {
		sign = "-"
	}
	return sign + s.integer.String()
}

func (s SignedInteger) Integer() int64 {
	if s.positive {
		return s.integer.Integer()
	}
	return -s.integer.Integer()
}

type SignedNumber struct {
	positive bool
	number   Number
}

func (s SignedNumber) String() string {
	sign := "+"
	if !s.positive {
		sign = "-"
	}
	return sign + s.number.String()
}

type Decimal struct{ runes }
type Octal struct{ runes }
type Hex struct{ runes }

func (d Decimal) Integer() int64 { return 0 } // TODO
func (o Octal) Integer() int64   { return 0 } // TODO
func (h Hex) Integer() int64     { return 0 } // TODO

type Float struct {
	integer  Decimal
	fraction *Decimal
	exponent *SignedInteger
}

func (s SignedNumber) Float() float64 {
	if s.positive {
		return s.number.Float()
	}
	return -s.number.Float()
}

func (f Float) String() string {
	return fmt.Sprintf("%s.%s E%s", f.integer, f.fraction, f.exponent)
}

// decimalLitToken // ( "1" … "9" ) { decimalDigit }
func (s *Scanner) number() Token {
	first := s.read()
	second := s.peek()

	if first == '0' && isDecimalDigit(second) {
		return s.octal()
	}
	if first == '0' && (second == 'x' || second == 'X') {
		s.read()
		return s.hex(first, second)
	}

	decimals := []rune{first}
	decimals = append(decimals, s.readWhile(isDecimalDigit)...)
	float := Float{
		integer: Decimal{decimals},
	}
	next := s.read()

	if next == dot {
		fraction := s.readWhile(isDecimalDigit)
		float.fraction = &Decimal{fraction}
		next = s.read()
	}

	if next == 'E' || next == 'e' {
		sign := s.read()
		if sign != '+' && sign != '-' {
			return errorf("expected exponent sign, but found %c", sign)
		}
		value := s.readWhile(isDecimalDigit)
		float.exponent = &SignedInteger{
			positive: sign == '+',
			integer:  Decimal{value},
		}
	}

	if float.fraction == nil && float.exponent == nil {
		return float.integer
	}
	return float
}

func (s *Scanner) octal() Token {
	value := s.readWhile(isOctalDigit)
	if isDecimalDigit(s.peek()) {
		return errorf("malformed octal constant %s%c", string(value), s.peek())
	}
	return Octal{value}
}

func (s *Scanner) hex(zero, x rune) Token {
	value := s.readWhile(isHexDigit)
	if len(value) == 0 {
		return errorf("malformed hex constant %c%c", zero, x)
	}
	return Hex{value}
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

// character classes

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

func and(fs ...runePredicate) runePredicate {
	return func(r rune) bool {
		for _, f := range fs {
			if !f(r) {
				return false
			}
		}
		return true
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
	semicolon   rune = ';'
)
