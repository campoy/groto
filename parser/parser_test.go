package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/campoy/groto/scanner"
)

func TestParseSyntax(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Syntax
		err  error
	}{
		{"good syntax", `syntax = "proto3";`, &Syntax{"proto3"}, nil},
		{"missing equal", `syntax "proto3";`, nil, errors.New(`expected punctuation '=', got "proto3"`)},
		{"bad text", `syntax = "proto2";`, nil, errors.New(`expected string "proto3", got "proto2"`)},
		{"missing semicolon", `syntax = "proto3"`, nil, errors.New(`expected punctuation ';', got "EOF"`)},
		{"missing quotes", `syntax = proto3;`, nil, errors.New(`expected string literal, got "proto3"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{Scanner: scanner.New(strings.NewReader(tt.in))}
			s, err := p.parseSyntax()
			if err != nil {
				if tt.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error %q; got %q", tt.err, err)
				}
				return
			}
			if s.Value != tt.out.Value {
				t.Fatalf("expected syntax %q; got %q", tt.out.Value, s.Value)
			}
		})
	}
}

func TestParseImport(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Import
		err  error
	}{
		{"good syntax", `import "path";`, &Import{Path: "path"}, nil},
		{"good public syntax", `import public "path";`, &Import{Path: "path", Modifier: "public"}, nil},
		{"good weak syntax", `import weak "path";`, &Import{Path: "path", Modifier: "weak"}, nil},
		{"bad modifier", `import bytes "path";`, nil, errors.New(`unexpected "bytes"`)},
		{"bad modifier keyword", `import enum "path";`, nil, errors.New(`unexpected keyword "enum"`)},
		{"bad import path", `import public path;`, nil, errors.New(`incorrect import path; got "path"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{Scanner: scanner.New(strings.NewReader(tt.in))}
			imp, err := p.parseImport()
			if err != nil {
				if tt.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error %q; got %q", tt.err, err)
				}
				return
			}
			if imp.Path != tt.out.Path {
				t.Fatalf("expected import Path %q; got %q", tt.out.Path, imp.Path)
			}
			if imp.Modifier != tt.out.Modifier {
				t.Fatalf("expect import modifier %q; got %q", tt.out.Modifier, imp.Modifier)
			}
		})
	}
}
