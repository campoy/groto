package token

import "strings"

type Token int

const (
	Illegal Token = iota
	EOF
	Comment

	FloatLiteral

	DecimalLiteral
	OctalLiteral
	HexLiteral

	StringLiteral
	Identifier
	FullIdentifier
	DottedIdentifer

	first_keyword
	Enum
	False
	Import
	Map
	Message
	Oneof
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
	True
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

	Equals
	Semicolon
	OpenParens
	CloseParens
	OpenBraces
	CloseBraces
	OpenBrackets
	CloseBracket
	OpenAngled
	CloseAngled
	Comma
	Minus
	Plus
)

var (
	keywords    = from(first_keyword, last_keyword)
	types       = from(first_type, last_type)
	punctuation = map[string]Token{
		"=": Equals,
		";": Semicolon,
		"(": OpenParens,
		")": CloseParens,
		"{": OpenBraces,
		"}": CloseBraces,
		"[": OpenBrackets,
		"]": CloseBracket,
		"<": OpenAngled,
		">": CloseAngled,
		",": Comma,
		"-": Minus,
		"+": Plus,
	}
)

func Keyword(s string) Token     { return keywords[s] }
func Type(s string) Token        { return types[s] }
func Punctuation(s string) Token { return punctuation[s] }

func from(a, b Token) map[string]Token {
	m := make(map[string]Token, b-a-1)
	for t := a + 1; t < b; t++ {
		m[strings.ToLower(t.String())] = t
	}
	return m
}
