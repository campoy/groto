// Copyright 2016 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package token

import (
	"fmt"
	"strings"
)

type Kind int

const (
	Illegal Kind = iota
	EOF
	Comment

	Identifier

	first_constant
	first_number
	DecimalLiteral
	FloatLiteral
	HexLiteral
	OctalLiteral
	last_number

	StringLiteral
	first_keyword
	False
	True
	last_constant

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

	Dot          // .
	Equals       // =
	Minus        // -
	Plus         // +
	Comma        // ,
	Semicolon    // ;
	OpenParen    // (
	CloseParen   // )
	OpenBrace    // {
	CloseBrace   // }
	OpenBracket  // [
	CloseBracket // ]
	OpenAngled   // <
	CloseAngled  // >

	last_kind
)

var (
	keywords    = from(first_keyword, last_keyword)
	types       = from(first_type, last_type)
	punctuation = map[string]Kind{
		".": Dot,
		"=": Equals,
		"-": Minus,
		"+": Plus,
		",": Comma,
		";": Semicolon,
		"(": OpenParen,
		")": CloseParen,
		"{": OpenBrace,
		"}": CloseBrace,
		"[": OpenBracket,
		"]": CloseBracket,
		"<": OpenAngled,
		">": CloseAngled,
	}
)

func Keyword(s string) Kind     { return keywords[s] }
func Type(s string) Kind        { return types[s] }
func Punctuation(s string) Kind { return punctuation[s] }

func IsKeyword(k Kind) bool  { return k > first_keyword && k < last_keyword }
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

func (k Kind) String() string {
	switch k {
	case Illegal:
		return "illegal"
	case EOF:
		return "end of file"
	case Comment:
		return "comment"
	case DecimalLiteral:
		return "decimal literal"
	case FloatLiteral:
		return "float literal"
	case HexLiteral:
		return "hex literal"
	case OctalLiteral:
		return "octal literal"
	case StringLiteral:
		return "string literal"
	case Identifier:
		return "identifier"

	// all keywords and types should show the exact text of the token when printed.
	case False:
		return "false"
	case True:
		return "true"
	case Enum:
		return "enum"
	case Import:
		return "import"
	case Map:
		return "map"
	case Message:
		return "message"
	case Oneof:
		return "oneof"
	case Option:
		return "option"
	case Package:
		return "package"
	case Public:
		return "public"
	case Repeated:
		return "repeated"
	case Reserved:
		return "reserved"
	case Returns:
		return "returns"
	case RPC:
		return "rpc"
	case Service:
		return "service"
	case Stream:
		return "stream"
	case Syntax:
		return "syntax"
	case To:
		return "to"
	case Weak:
		return "weak"
	case Bytes:
		return "bytes"
	case Double:
		return "double"
	case Float:
		return "float"
	case Bool:
		return "bool"
	case Fixed32:
		return "fixed32"
	case Fixed64:
		return "fixed64"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Sfixed32:
		return "sfixed32"
	case Sfixed64:
		return "sfixed64"
	case Sint32:
		return "sint32"
	case Sint64:
		return "sint64"
	case String:
		return "string"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"

	case Dot:
		return "'.'"
	case Equals:
		return "'='"
	case Minus:
		return "'-'"
	case Plus:
		return "'+'"
	case Comma:
		return "','"
	case Semicolon:
		return "';'"
	case OpenParen:
		return "'('"
	case CloseParen:
		return "')'"
	case OpenBrace:
		return "'{'"
	case CloseBrace:
		return "'}'"
	case OpenBracket:
		return "'['"
	case CloseBracket:
		return "']'"
	case OpenAngled:
		return "'<'"
	case CloseAngled:
		return "'>'"
	default:
		return fmt.Sprintf("unkown token kind %d", k)
	}
}
