package scanner

import (
	"reflect"
	"strings"
	"testing"
)

func float(integer, fraction, sign, exp string) Token {
	f := Float{
		integer: Decimal{stringToken(integer)},
	}
	if fraction != "" {
		f.fraction = &Decimal{stringToken(fraction)}
	}
	if exp != "" {
		if sign != "+" && sign != "-" {
			panic("bad sign in test " + sign)
		}
		f.exponent = &SignedInteger{
			positive: sign == "+",
			integer:  Decimal{stringToken(exp)},
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
			Identifier{"x"},
		}},
		{"longer ident", "counter", []Token{
			Identifier{"counter"},
		}},
		{"two identifiers", "a b ", []Token{
			Identifier{"a"}, Identifier{"b"},
		}},
		{"decimal numbers", "0 1 20 30000", []Token{
			Decimal{"0"}, Decimal{"1"}, Decimal{"20"}, Decimal{"30000"},
		}},
		{"octal numbers", "01 020", []Token{
			Octal{"1"}, Octal{"20"},
		}},
		{"hex numbers", "0x1 0xA2F", []Token{
			Hex{"1"}, Hex{"A2F"},
		}},
		{"float numbers", "0.1E+2 1.2 1.3E-10 4e+5", []Token{
			float("0", "1", "+", "2"),
			float("1", "2", "", ""),
			float("1", "3", "-", "10"),
			float("4", "", "+", "5"),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(strings.NewReader(tt.in))
			rem := tt.out
			for {
				tok := s.Scan()
				if tok == EOF {
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
