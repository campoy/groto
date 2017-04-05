package scanner

import (
	"strings"
	"testing"

	"github.com/campoy/groto/token"
)

func TestScanner(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []Token
	}{
		{"empty string", "", nil},
		{"one letter ident", "x", []Token{
			{token.Identifier, "x"},
		}},
		{"longer ident", "counter", []Token{
			{token.Identifier, "counter"},
		}},
		{"full identifier", "one.two.three", []Token{
			{token.Identifier, "one"}, {token.Dot, ""},
			{token.Identifier, "two"}, {token.Dot, ""},
			{token.Identifier, "three"},
		}},
		{"two identifiers", "a b ", []Token{
			{token.Identifier, "a"},
			{token.Identifier, "b"},
		}},
		{"decimal numbers", "0 1 20 30000", []Token{
			{token.DecimalLiteral, "0"},
			{token.DecimalLiteral, "1"},
			{token.DecimalLiteral, "20"},
			{token.DecimalLiteral, "30000"},
		}},
		{"octal numbers", "01 020 019", []Token{
			{token.OctalLiteral, "01"},
			{token.OctalLiteral, "020"},
			{token.Illegal, "019"},
		}},
		{"hex numbers", "0x1 0XA2F 0x", []Token{
			{token.HexLiteral, "0x1"},
			{token.HexLiteral, "0XA2F"},
			{token.Illegal, "0x"},
		}},
		{"float numbers", "0.1E+2 1.2 1.3E-10 4e+5 4e.5", []Token{
			{token.FloatLiteral, "0.1E+2"},
			{token.FloatLiteral, "1.2"},
			{token.FloatLiteral, "1.3E-10"},
			{token.FloatLiteral, "4e+5"},
			{token.Illegal, "4e."}, {token.DecimalLiteral, "5"},
		}},
		{"signed numbers", "+0 -010 -0xfff +0.5 -1", []Token{
			{token.Plus, ""}, {token.DecimalLiteral, "0"},
			{token.Minus, ""}, {token.OctalLiteral, "010"},
			{token.Minus, ""}, {token.HexLiteral, "0xfff"},
			{token.Plus, ""}, {token.FloatLiteral, "0.5"},
			{token.Minus, ""}, {token.DecimalLiteral, "1"},
		}},
		{"double quote strings", `"" "hello" "hello\" there"`, []Token{
			{token.StringLiteral, `""`},
			{token.StringLiteral, `"hello"`},
			{token.StringLiteral, `"hello\" there"`},
		}},
		{"single quote strings", `'' 'hello' 'hello\' there'`, []Token{
			{token.StringLiteral, `''`},
			{token.StringLiteral, `'hello'`},
			{token.StringLiteral, `'hello\' there'`},
		}},
		{"booleans", `true false`, []Token{
			{token.True, ""},
			{token.False, ""},
		}},
		{"keywords, booleans, and idents", `import false hello bytes`, []Token{
			{token.Import, ""},
			{token.False, ""},
			{token.Identifier, "hello"},
			{token.Bytes, ""},
		}},
		{"punctuation", `(){};=`, []Token{
			{token.OpenParens, ""},
			{token.CloseParens, ""},
			{token.OpenBraces, ""},
			{token.CloseBraces, ""},
			{token.Semicolon, ""},
			{token.Equals, ""},
		}},
		{"comments", "text // a comment\nimport", []Token{
			{token.Identifier, "text"},
			{token.Comment, "// a comment"},
			{token.Import, ""},
		}},
		{"unexpected characters", "#blessed", []Token{
			{token.Illegal, "#"},
			{token.Identifier, "blessed"},
		}},
		{"not a comment", "/* badcomment", []Token{
			{token.Illegal, "/*"},
			{token.Identifier, "badcomment"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(strings.NewReader(tt.in))
			for i, want := range tt.out {
				got := s.Scan()
				if got.Kind == token.EOF {
					if len(tt.out) > i+1 {
						t.Fatalf("remaining tokens: %v", tt.out[i+1:])
					}
					break
				}
				if want.Kind != got.Kind {
					t.Errorf("token[%d] expected token of kind %v, got %v", i, want.Kind, got.Kind)
				}
				if want.Text != got.Text {
					t.Errorf("token[%d] expected token value %q, got %q", i, want.Text, got.Text)
				}
			}
			if tok := s.Scan(); tok.Kind != token.EOF {
				t.Errorf("unexpected token %v %s", tok.Kind, tok.Text)
			}
		})
	}
}
