package parser

import (
	"errors"
	"strings"
	"testing"

	"reflect"

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
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
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
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
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
				t.Fatalf("expected import path %q; got %q", tt.out.Path, imp.Path)
			}
			if imp.Modifier != tt.out.Modifier {
				t.Fatalf("expected import modifier %q; got %q", tt.out.Modifier, imp.Modifier)
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

		{"single identifier", `package foo;`, &Package{scanner.Identifier{scanner.Runes("foo")}}, nil},
		{"full identifer", `package com.example.foo;`, &Package{scanner.FullIdentifier{
			[]scanner.Identifier{{scanner.Runes("com")}, {scanner.Runes("example")}, {scanner.Runes("foo")}}},
		}, nil},
		{"bad identifier", `package "foo";`, nil, errors.New("expected package identifier, got \"foo\"")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			pkg, err := p.parsePackage()
			if err != nil {
				if tt.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error %q; got %q", tt.err, err)
				}
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

		{"good syntax string", `option java_package = "com.example.foo";`,
			&Option{
				Name:  scanner.Identifier{scanner.Runes("java_package")},
				Value: scanner.String{scanner.Runes("com.example.foo")},
			}, nil},
		{"good syntax full identifer", `option java_package = foo.bar;`,
			&Option{
				Name:  scanner.Identifier{scanner.Runes("java_package")},
				Value: scanner.FullIdentifier{[]scanner.Identifier{{scanner.Runes("foo")}, {scanner.Runes("bar")}}},
			}, nil},
		{"good syntax integer", `option options.number = 42;`,
			&Option{
				Name:  scanner.FullIdentifier{[]scanner.Identifier{{scanner.Runes("options")}, {scanner.Runes("number")}}},
				Value: scanner.Decimal{scanner.Runes("42")},
			}, nil},
		{"bad syntax", `option java_package = syntax;`, nil, errors.New(`unsupported option value scanner.Keyword "syntax"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{s: scanner.New(strings.NewReader(tt.in))}
			opt, err := p.parseOption()
			if err != nil {
				if tt.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error %q; got %q", tt.err, err)
				}
				return
			}
			if !reflect.DeepEqual(opt.Name, tt.out.Name) {
				t.Fatalf("expected option name %q; got %q", tt.out.Name, opt.Name)
			}
			if !reflect.DeepEqual(opt.Value, tt.out.Value) {
				t.Fatalf("expected option value %q; got %q", tt.out.Value, opt.Value)
			}
		})
	}
}
