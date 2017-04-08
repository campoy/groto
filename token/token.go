package token

import "strings"

type Kind int

const (
	Illegal Kind = iota
	EOF
	Comment

	first_constant
	first_number
	DecimalLiteral
	FloatLiteral
	HexLiteral
	OctalLiteral
	last_number

	StringLiteral
	False
	True
	last_constant

	Identifier

	first_keyword
	Enum
	Import
	Map
	Message
	Oneof
	Option
	Package
	Public
	Repeated
	Reserved
	Returns
	RPC
	Service
	Stream
	Syntax
	To
	Weak
	last_keyword

	first_type
	Bytes
	Double
	Float
	first_key_type
	Bool
	Fixed32
	Fixed64
	Int32
	Int64
	Sfixed32
	Sfixed64
	Sint32
	Sint64
	String
	Uint32
	Uint64
	last_type

	CloseAngled
	CloseBraces
	CloseBracket
	CloseParens
	Comma
	Dot
	Equals
	Minus
	OpenAngled
	OpenBraces
	OpenBrackets
	OpenParens
	Plus
	Semicolon
)

var (
	keywords    = from(first_keyword, last_keyword)
	types       = from(first_type, last_type)
	punctuation = map[string]Kind{
		">": CloseAngled,
		"}": CloseBraces,
		"]": CloseBracket,
		")": CloseParens,
		",": Comma,
		".": Dot,
		"=": Equals,
		"-": Minus,
		"<": OpenAngled,
		"{": OpenBraces,
		"[": OpenBrackets,
		"(": OpenParens,
		"+": Plus,
		";": Semicolon,
	}
)

func Keyword(s string) Kind     { return keywords[s] }
func Type(s string) Kind        { return types[s] }
func Punctuation(s string) Kind { return punctuation[s] }

func IsConstant(k Kind) bool { return k > first_constant && k < last_constant }
func IsNumber(k Kind) bool   { return k > first_number && k < last_number }
func IsType(k Kind) bool     { return k > first_type && k < last_type }
func IsKeyType(k Kind) bool  { return k > first_key_type && k < last_type }

func from(a, b Kind) map[string]Kind {
	m := make(map[string]Kind, b-a-1)
	for t := a + 1; t < b; t++ {
		m[strings.ToLower(t.String())] = t
	}
	return m
}
