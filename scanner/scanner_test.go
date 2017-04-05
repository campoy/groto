package scanner

import (
	"strings"
	"testing"

	"github.com/campoy/groto/token"
)

type tokenValue struct {
	tok   token.Token
	value string
}

func TestScanner(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []tokenValue
	}{
		{"empty string", "", nil},
		{"one letter ident", "x", []tokenValue{
			{token.Identifier, "x"},
		}},
		{"longer ident", "counter", []tokenValue{
			{token.Identifier, "counter"},
		}},
		{"full identifier", "one.two.three", []tokenValue{
			{token.FullIdentifier, "one.two.three"},
		}},
		{"two identifiers", "a b ", []tokenValue{
			{token.Identifier, "a"},
			{token.Identifier, "b"},
		}},
		{"decimal numbers", "0 1 20 30000", []tokenValue{
			{token.DecimalLiteral, "0"},
			{token.DecimalLiteral, "1"},
			{token.DecimalLiteral, "20"},
			{token.DecimalLiteral, "30000"},
		}},
		{"octal numbers", "01 020 019", []tokenValue{
			{token.OctalLiteral, "01"},
			{token.OctalLiteral, "020"},
			{token.Illegal, "019"},
		}},
		{"hex numbers", "0x1 0XA2F 0x", []tokenValue{
			{token.HexLiteral, "0x1"},
			{token.HexLiteral, "0XA2F"},
			{token.Illegal, "0x"},
		}},
		{"float numbers", "0.1E+2 1.2 1.3E-10 4e+5 4e.5", []tokenValue{
			{token.FloatLiteral, "0.1E+2"},
			{token.FloatLiteral, "1.2"},
			{token.FloatLiteral, "1.3E-10"},
			{token.FloatLiteral, "4e+5"},
			{token.Illegal, "4e."}, {token.DecimalLiteral, "5"},
		}},
		{"signed numbers", "+0 -010 -0xfff +0.5 -1", []tokenValue{
			{token.Plus, ""}, {token.DecimalLiteral, "0"},
			{token.Minus, ""}, {token.OctalLiteral, "010"},
			{token.Minus, ""}, {token.HexLiteral, "0xfff"},
			{token.Plus, ""}, {token.FloatLiteral, "0.5"},
			{token.Minus, ""}, {token.DecimalLiteral, "1"},
		}},
		{"double quote strings", `"" "hello" "hello\" there"`, []tokenValue{
			{token.StringLiteral, `""`},
			{token.StringLiteral, `"hello"`},
			{token.StringLiteral, `"hello\" there"`},
		}},
		{"single quote strings", `'' 'hello' 'hello\' there'`, []tokenValue{
			{token.StringLiteral, `''`},
			{token.StringLiteral, `'hello'`},
			{token.StringLiteral, `'hello\' there'`},
		}},
		{"booleans", `true false`, []tokenValue{
			{token.True, ""},
			{token.False, ""},
		}},
		{"keywords, booleans, and idents", `import false hello bytes`, []tokenValue{
			{token.Import, ""},
			{token.False, ""},
			{token.Identifier, "hello"},
			{token.Bytes, ""},
		}},
		{"punctuation", `(){};=`, []tokenValue{
			{token.OpenParens, ""},
			{token.CloseParens, ""},
			{token.OpenBraces, ""},
			{token.CloseBraces, ""},
			{token.Semicolon, ""},
			{token.Equals, ""},
		}},
		{"comments", "text // a comment\nimport", []tokenValue{
			{token.Identifier, "text"},
			{token.Comment, "// a comment"},
			{token.Import, ""},
		}},
		{"unexpected characters", "#blessed", []tokenValue{
			{token.Illegal, "#"},
			{token.Identifier, "blessed"},
		}},
		{"not a comment", "/* badcomment", []tokenValue{
			{token.Illegal, "/*"},
			{token.Identifier, "badcomment"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(strings.NewReader(tt.in))
			for i, want := range tt.out {
				tok, value := s.Scan()
				if tok == token.EOF {
					if len(tt.out) > i+1 {
						t.Fatalf("remaining tokens: %v", tt.out[i+1:])
					}
					break
				}
				if want.tok != tok {
					t.Errorf("token[%d] expected token type %v, got %v", i, want.tok, tok)
				}
				if want.value != string(value) {
					t.Errorf("token[%d] expected token value %q, got %q", i, want.value, string(value))
				}
			}
			if tok, value := s.Scan(); tok != token.EOF {
				t.Errorf("unexpected token %v %s", tok, string(value))
			}
		})
	}
}
