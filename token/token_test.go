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

import "testing"

func TestConstructors(t *testing.T) {
	tests := []struct {
		text string
		f    func(string) Kind
		kind Kind
	}{
		{"false", Keyword, False},
		{"true", Keyword, True},
		{"enum", Keyword, Enum},
		{"import", Keyword, Import},
		{"map", Keyword, Map},
		{"message", Keyword, Message},
		{"oneof", Keyword, Oneof},
		{"option", Keyword, Option},
		{"package", Keyword, Package},
		{"public", Keyword, Public},
		{"repeated", Keyword, Repeated},
		{"reserved", Keyword, Reserved},
		{"returns", Keyword, Returns},
		{"rpc", Keyword, RPC},
		{"service", Keyword, Service},
		{"stream", Keyword, Stream},
		{"syntax", Keyword, Syntax},
		{"to", Keyword, To},
		{"weak", Keyword, Weak},

		{"bytes", Type, Bytes},
		{"double", Type, Double},
		{"float", Type, Float},
		{"bool", Type, Bool},
		{"fixed32", Type, Fixed32},
		{"fixed64", Type, Fixed64},
		{"int32", Type, Int32},
		{"int64", Type, Int64},
		{"sfixed32", Type, Sfixed32},
		{"sfixed64", Type, Sfixed64},
		{"sint32", Type, Sint32},
		{"sint64", Type, Sint64},
		{"string", Type, String},
		{"uint32", Type, Uint32},
		{"uint64", Type, Uint64},

		{".", Punctuation, Dot},
		{"=", Punctuation, Equals},
		{"-", Punctuation, Minus},
		{"+", Punctuation, Plus},
		{",", Punctuation, Comma},
		{";", Punctuation, Semicolon},
		{"(", Punctuation, OpenParen},
		{")", Punctuation, CloseParen},
		{"{", Punctuation, OpenBrace},
		{"}", Punctuation, CloseBrace},
		{"[", Punctuation, OpenBracket},
		{"]", Punctuation, CloseBracket},
		{"<", Punctuation, OpenAngled},
		{">", Punctuation, CloseAngled},
	}
	for _, tt := range tests {
		if tt.f(tt.text) != tt.kind {
			t.Errorf("text %s should be matched with kind %s", tt.text, tt.kind)
		}
	}
}

type predicate string

const (
	isKeyword  = "keyword"
	isConstant = "constant"
	isNumber   = "number"
	isType     = "type"
	isKeyType  = "key type"
)

var funcs = map[predicate]func(Kind) bool{
	isKeyword:  IsKeyword,
	isConstant: IsConstant,
	isNumber:   IsNumber,
	isType:     IsType,
	isKeyType:  IsKeyType,
}

func only(ps ...predicate) map[predicate]bool {
	m := make(map[predicate]bool)
	for _, p := range ps {
		m[p] = true
	}
	return m
}

func TestIsKeyword(t *testing.T) {
	tests := []struct {
		kind       Kind
		predicates map[predicate]bool
	}{
		{EOF, only()},
		{Comment, only()},
		{Identifier, only()},
		{DecimalLiteral, only(isNumber, isConstant)},
		{FloatLiteral, only(isNumber, isConstant)},
		{HexLiteral, only(isNumber, isConstant)},
		{OctalLiteral, only(isNumber, isConstant)},
		{StringLiteral, only(isConstant)},
		{False, only(isKeyword, isConstant)},
		{True, only(isKeyword, isConstant)},
		{Enum, only(isKeyword)},
		{Import, only(isKeyword)},
		{Map, only(isKeyword)},
		{Message, only(isKeyword)},
		{Oneof, only(isKeyword)},
		{Option, only(isKeyword)},
		{Package, only(isKeyword)},
		{Public, only(isKeyword)},
		{Repeated, only(isKeyword)},
		{Reserved, only(isKeyword)},
		{Returns, only(isKeyword)},
		{RPC, only(isKeyword)},
		{Service, only(isKeyword)},
		{Stream, only(isKeyword)},
		{Syntax, only(isKeyword)},
		{To, only(isKeyword)},
		{Weak, only(isKeyword)},
		{Bytes, only(isType)},
		{Double, only(isType)},
		{Float, only(isType)},
		{Bool, only(isType, isKeyType)},
		{Fixed32, only(isType, isKeyType)},
		{Fixed64, only(isType, isKeyType)},
		{Int32, only(isType, isKeyType)},
		{Int64, only(isType, isKeyType)},
		{Sfixed32, only(isType, isKeyType)},
		{Sfixed64, only(isType, isKeyType)},
		{Sint32, only(isType, isKeyType)},
		{Sint64, only(isType, isKeyType)},
		{String, only(isType, isKeyType)},
		{Uint32, only(isType, isKeyType)},
		{Uint64, only(isType, isKeyType)},
		{Dot, only()},
		{Equals, only()},
		{Minus, only()},
		{Plus, only()},
		{Comma, only()},
		{Semicolon, only()},
		{OpenParen, only()},
		{CloseParen, only()},
		{OpenBrace, only()},
		{CloseBrace, only()},
		{OpenBracket, only()},
		{CloseBracket, only()},
		{OpenAngled, only()},
		{CloseAngled, only()},
	}

	for _, tt := range tests {
		for p, f := range funcs {
			if f(tt.kind) != tt.predicates[p] {
				if tt.predicates[p] {
					t.Errorf("%s should be a %s, but it's not", tt.kind, p)
				} else {
					t.Errorf("%s should not be a %s, but it is", tt.kind, p)
				}
			}
		}
	}
}
