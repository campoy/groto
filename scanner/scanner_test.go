package scanner

import (
	"reflect"
	"strings"
	"testing"
)

func float(integer, fraction, sign, exp string) Float {
	f := Float{
		integer: Decimal{Runes(integer)},
	}
	if fraction != "" {
		f.fraction = &Decimal{Runes(fraction)}
	}
	if exp != "" {
		if sign != "+" && sign != "-" {
			panic("bad sign in test " + sign)
		}
		f.exponent = &SignedInteger{
			positive: sign == "+",
			integer:  Decimal{Runes(exp)},
		}
	}
	return f
}

func TestScanner(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []Token
	}{
		{"empty string", "", nil},
		{"one letter ident", "x", []Token{
			Identifier{Runes("x")},
		}},
		{"longer ident", "counter", []Token{
			Identifier{Runes("counter")},
		}},
		{"full identifier", "one.two.three", []Token{
			FullIdentifier{[]Identifier{{Runes("one")}, {Runes("two")}, {Runes("three")}}},
		}},
		{"two identifiers", "a b ", []Token{
			Identifier{Runes("a")}, Identifier{Runes("b")},
		}},
		{"decimal numbers", "0 1 20 30000", []Token{
			Decimal{Runes("0")}, Decimal{Runes("1")}, Decimal{Runes("20")}, Decimal{Runes("30000")},
		}},
		{"octal numbers", "01 020", []Token{
			Octal{Runes("1")}, Octal{Runes("20")},
		}},
		{"hex numbers", "0x1 0xA2F", []Token{
			Hex{Runes("1")}, Hex{Runes("A2F")},
		}},
		{"float numbers", "0.1E+2 1.2 1.3E-10 4e+5", []Token{
			float("0", "1", "+", "2"),
			float("1", "2", "", ""),
			float("1", "3", "-", "10"),
			float("4", "", "+", "5"),
		}},
		{"signed numbers", "+0 -010 -0xfff +0.5 -1", []Token{
			SignedInteger{positive: true, integer: Decimal{Runes("0")}},
			SignedInteger{positive: false, integer: Octal{Runes("10")}},
			SignedInteger{positive: false, integer: Hex{Runes("fff")}},
			SignedNumber{positive: true, number: float("0", "5", "", "")},
			SignedInteger{positive: false, integer: Decimal{Runes("1")}},
		}},
		{"double quote strings", `"" "hello" "hello\" there"`, []Token{
			String{}, String{Runes(`hello`)}, String{Runes(`hello\" there`)},
		}},
		{"single quote strings", `'' 'hello' 'hello\' there'`, []Token{
			String{}, String{Runes(`hello`)}, String{Runes(`hello\' there`)},
		}},
		{"booleans", `true false`, []Token{
			Boolean{true}, Boolean{false},
		}},
		{"keywords, booleans, and idents", `import false hello bytes`, []Token{
			Keyword{Runes("import")}, Boolean{false}, Identifier{Runes("hello")}, Type{Runes("bytes")},
		}},
		{"punctuation", `(){};=`, []Token{
			Punctuation{OpenParens}, Punctuation{CloseParens},
			Punctuation{OpenBrace}, Punctuation{CloseBrace},
			Punctuation{Semicolon}, Punctuation{Equal},
		}},
		{"comments", "text // a comment\nimport", []Token{
			Identifier{Runes("text")}, Comment{Runes("a comment")}, Keyword{Runes("import")},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(strings.NewReader(tt.in))
			rem := tt.out
			for {
				tok := s.Scan()
				if IsEOF(tok) {
					break
				}
				if len(rem) == 0 {
					t.Errorf("unexpected token %+v", tok)
					return
				}
				want := rem[0]
				rem = rem[1:]
				if !reflect.DeepEqual(want, tok) {
					t.Errorf("expected %#v, got %#v", want, tok)
				}
			}
		})
	}
}
