package scanner

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func New(r io.Reader) *Scanner {
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

func IsEOF(tok Token) bool {
	_, ok := tok.(EOF)
	return ok
}

type EOF struct{}

func (tok EOF) String() string { return "EOF" }

type Error struct{ msg string }

func errorf(format string, args ...interface{}) Token { return Error{fmt.Sprintf(format, args...)} }
func (err Error) String() string                      { return err.msg }
func (err Error) Error() string                       { return err.msg }

type Punctuation struct{ Value rune }

func (tok Punctuation) String() string { return string(tok.Value) }

var (
	Dot         rune = '.'
	Equal       rune = '='
	Semicolon   rune = ';'
	OpenBrace   rune = '{'
	CloseBrace  rune = '}'
	OpenParens  rune = '('
	CloseParens rune = ')'
)

var punctuation = map[rune]bool{
	Dot:         true,
	Equal:       true,
	Semicolon:   true,
	OpenBrace:   true,
	CloseBrace:  true,
	OpenParens:  true,
	CloseParens: true,
}

func (s *Scanner) Scan() Token {
	s.readWhile(isSpace)

	r := s.peek()
	switch {
	case r == eof:
		return EOF{}
	case isLetter(r):
		return s.identifier()
	case isDecimalDigit(r):
		return s.number()
	case r == quote || r == doubleQuote:
		return s.string()
	case r == '+' || r == '-':
		return s.signedNumber()
	case r == '/':
		return s.comment()
	case punctuation[r]:
		s.read()
		return Punctuation{r}
	default:
		return errorf("unexpected character %c", r)
	}
}

// IDENTIFIERS

type runes []rune

func (r runes) String() string { return string(r) }

type Identifier struct{ runes }

type FullIdentifier struct {
	Identfiers []Identifier
}

func (i FullIdentifier) String() string {
	var names []string
	for _, n := range i.Identfiers {
		names = append(names, n.String())
	}
	return strings.Join(names, ".")
}

type Boolean struct{ value bool }

func (b Boolean) String() string {
	if b.value {
		return "true"
	}
	return "false"
}

type Keyword struct{ runes }

var keywords = map[string]bool{
	"enum":    true,
	"import":  true,
	"message": true,
	"option":  true,
	"package": true,
	"public":  true,
	"service": true,
	"syntax":  true,
	"weak":    true,
}

type Type struct{ runes }

var types = map[string]bool{
	"bool":     true,
	"bytes":    true,
	"double":   true,
	"fixed32":  true,
	"fixed64":  true,
	"float":    true,
	"int32":    true,
	"int64":    true,
	"sfixed32": true,
	"sfixed64": true,
	"sint32":   true,
	"sint64":   true,
	"string":   true,
	"uint32":   true,
	"uint64":   true,
}

func (s *Scanner) identifier() Token {
	value := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))

	if s.peek() == dot {
		return s.fullIdentifier(value)
	}

	switch text := string(value); {
	case text == "true":
		return Boolean{true}
	case text == "false":
		return Boolean{false}
	case keywords[text]:
		return Keyword{value}
	case types[text]:
		return Type{value}
	default:
		return Identifier{value}
	}
}

func (s *Scanner) fullIdentifier(start []rune) Token {
	idents := []Identifier{{start}}
	for s.peek() == dot {
		s.read()
		value := s.readWhile(or(isLetter, isDecimalDigit, equals(underscore)))
		idents = append(idents, Identifier{value})
	}
	return FullIdentifier{idents}
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

// COMMENTS

type Comment struct{ runes }

func (s *Scanner) comment() Token {
	s.read()
	second := s.read()
	if second != '/' {
		s.unread()
		return errorf("unexpected / that is not part of a comment")
	}

	s.readWhile(isSpace)
	text := s.readUntil(equals('\n'))
	return Comment{text}
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

func (f Float) Float() float64 { return 0 } // TODO

func (s *Scanner) signedNumber() Token {
	positive := s.read() == '+'
	tok := s.number()
	if i, ok := tok.(Integer); ok {
		return SignedInteger{positive, i}
	}
	return SignedNumber{positive, tok.(Number)}
}

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
