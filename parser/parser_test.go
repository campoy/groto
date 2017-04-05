package parser

import (
	"errors"
	"strings"
	"testing"

	"reflect"

	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

func make(kind token.Kind, text string) scanner.Token {
	return scanner.Token{Kind: kind, Text: text}
}

func ptr(tok scanner.Token) *scanner.Token { return &tok }

func checkErrors(t *testing.T, want, got error) bool {
	switch {
	case want == nil && got == nil:
		return true
	case want == nil && got != nil:
		t.Fatalf("unexpected error: %v", got)
	case want != nil && got == nil:
		t.Fatalf("expected error %v, got nothing", want)
	case want.Error() != got.Error():
		t.Fatalf("expected error %q; got %q", want, got)
	}
	return false
}

func TestParseSyntax(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Syntax
		err  error
	}{
		{name: "good syntax", in: `syntax = "proto3";`,
			out: &Syntax{make(token.StringLiteral, `"proto3"`)},
		},
		{name: "missing equal", in: `syntax "proto3";`,
			err: errors.New(`expected '=', got "proto3" instead`),
		},
		{name: "bad text", in: `syntax = "proto2";`,
			err: errors.New(`expected literal string "proto3", got "proto2" instead`),
		},
		{name: "missing semicolon", in: `syntax = "proto3"`,
			err: errors.New(`missing semicolon at the end of the syntax statement`),
		},
		{name: "missing quotes", in: `syntax = proto3;`,
			err: errors.New(`expected literal string "proto3", got a Identifier instead`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			var s Syntax
			err := s.parse(p)
			if !checkErrors(t, tt.err, err) {
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
		{name: "good syntax", in: `import "path";`,
			out: &Import{Path: make(token.StringLiteral, `"path"`)},
		},
		{name: "good public syntax", in: `import public "path";`,
			out: &Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Public, "")},
		},
		{name: "good weak syntax", in: `import weak "path";`,
			out: &Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Weak, "")},
		},
		{name: "bad modifier", in: `import bytes "path";`,
			err: errors.New(`expected imported package name, got bytes`),
		},
		{name: "bad modifier keyword", in: `import enum "path";`,
			err: errors.New(`expected imported package name, got enum`),
		},
		{name: "bad import path", in: `import public path;`,
			err: errors.New(`expected imported package name, got identifier (path)`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			var imp Import
			err := imp.parse(p)
			if !checkErrors(t, tt.err, err) {
				return
			}
			if imp.Path != tt.out.Path {
				t.Fatalf("expected import path %q; got %q", tt.out.Path, imp.Path)
			}
			if imp.Modifier != tt.out.Modifier {
				t.Fatalf("expected import modifier %v; got %v", tt.out.Modifier, imp.Modifier)
			}
		})
	}
}

func TestParsePackage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Package
		err  error
	}{

		{name: "single identifier", in: `package foo;`,
			out: &Package{FullIdentifier{[]scanner.Token{
				make(token.Identifier, "foo"),
			}}},
		},
		{name: "full identifer", in: `package com.example.foo;`,
			out: &Package{FullIdentifier{[]scanner.Token{
				make(token.Identifier, "com"),
				make(token.Identifier, "examples"),
				make(token.Identifier, "foo"),
			}}},
		},
		{name: "bad identifier", in: `package "foo";`,
			err: errors.New("expected package identifier, got \"foo\""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			var pkg Package
			err := pkg.parse(p)
			if !checkErrors(t, tt.err, err) {
				return
			}
			if !reflect.DeepEqual(pkg.Identifier, tt.out.Identifier) {
				t.Fatalf("expected package identifer %q; got %q", tt.out.Identifier, pkg.Identifier)
			}
		})
	}
}

func TestParseOption(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Option
		err  error
	}{

		{name: "good syntax string", in: `option java_package = "com.example.foo";`,
			out: &Option{
				Name:  FullIdentifier{[]scanner.Token{make(token.Identifier, "java_package")}},
				Value: Constant{make(token.StringLiteral, `"com.example.foo"`)},
			},
		},
		{name: "good syntax full identifer", in: `option java_package = foo.bar;`,
			out: &Option{
				Name: FullIdentifier{[]scanner.Token{make(token.Identifier, "java_package")}},
				Value: Constant{FullIdentifier{[]scanner.Token{
					make(token.Identifier, "foo"),
					make(token.Identifier, "bar"),
				}}},
			},
		},
		{name: "good syntax integer", in: `option options.number = 42;`,
			out: &Option{
				Name: FullIdentifier{[]scanner.Token{
					make(token.Identifier, "options"),
					make(token.Identifier, "number"),
				}},
				Value: Constant{make(token.DecimalLiteral, "42")},
			},
		},
		{name: "bad syntax", in: `option java_package = syntax;`,
			err: errors.New(`unsupported option value scanner.Keyword "syntax"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			var opt Option
			err := opt.parse(p)
			checkErrors(t, tt.err, err)
			if !reflect.DeepEqual(opt.Name, tt.out.Name) {
				t.Fatalf("expected option name %q; got %q", tt.out.Name, opt.Name)
			}
			if !reflect.DeepEqual(opt.Value, tt.out.Value) {
				t.Fatalf("expected option value %q; got %q", tt.out.Value, opt.Value)
			}
		})
	}
}
