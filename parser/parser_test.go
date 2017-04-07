package parser

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"

	"reflect"

	"github.com/Sirupsen/logrus"
	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
	"github.com/kr/pretty"
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

func TestParse(t *testing.T) {
	logf = logrus.Infof

	tests := map[string][]struct {
		name   string
		in     string
		target parser
		out    parser
		err    error
	}{
		"Syntax": {
			{name: "good syntax", in: `syntax = "proto3";`, target: new(Syntax),
				out: &Syntax{make(token.StringLiteral, `"proto3"`)},
			},
			{name: "missing equal", in: `syntax "proto3";`, target: new(Syntax),
				err: errors.New(`expected '=', got "proto3" instead`),
			},
			{name: "bad text", in: `syntax = "proto2";`, target: new(Syntax),
				err: errors.New(`expected literal string "proto3", got "proto2" instead`),
			},
			{name: "missing semicolon", in: `syntax = "proto3"`, target: new(Syntax),
				err: errors.New(`missing semicolon at the end of the syntax statement`),
			},
			{name: "missing quotes", in: `syntax = proto3;`, target: new(Syntax),
				err: errors.New(`expected literal string "proto3", got a Identifier instead`),
			},
		},

		"Import": {
			{name: "good syntax", in: `import "path";`, target: new(Import),
				out: &Import{Path: make(token.StringLiteral, `"path"`)},
			},
			{name: "good public syntax", in: `import public "path";`, target: new(Import),
				out: &Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Public, "")},
			},
			{name: "good weak syntax", in: `import weak "path";`, target: new(Import),
				out: &Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Weak, "")},
			},
			{name: "bad modifier", in: `import bytes "path";`, target: new(Import),
				err: errors.New(`expected imported package name, got bytes`),
			},
			{name: "bad modifier keyword", in: `import enum "path";`, target: new(Import),
				err: errors.New(`expected imported package name, got enum`),
			},
			{name: "bad import path", in: `import public path;`, target: new(Import),
				err: errors.New(`expected imported package name, got identifier (path)`),
			},
		},

		"Package": {
			{name: "single identifier", in: `package foo;`, target: new(Package),
				out: &Package{FullIdentifier{[]scanner.Token{
					make(token.Identifier, "foo"),
				}}},
			},
			{name: "full identifer", in: `package com.example.foo;`, target: new(Package),
				out: &Package{FullIdentifier{[]scanner.Token{
					make(token.Identifier, "com"),
					make(token.Identifier, "example"),
					make(token.Identifier, "foo"),
				}}},
			},
			{name: "bad identifier", in: `package "foo";`, target: new(Package),
				err: errors.New("expected identifier, got string literal (\"foo\")"),
			},
		},

		"Option": {
			{name: "good syntax string", in: `option java_package = "com.example.foo";`, target: new(Options),
				out: &Options{{
					Name:  &FullIdentifier{[]scanner.Token{make(token.Identifier, "java_package")}},
					Value: Constant{make(token.StringLiteral, `"com.example.foo"`)},
				}},
			},
			{name: "good syntax full identifer", in: `option java_package = foo.bar;`, target: new(Options),
				out: &Options{{
					Name: &FullIdentifier{[]scanner.Token{make(token.Identifier, "java_package")}},
					Value: Constant{FullIdentifier{[]scanner.Token{
						make(token.Identifier, "foo"),
						make(token.Identifier, "bar"),
					}}},
				}},
			},
			{name: "good syntax integer", in: `option options.number = 42;`, target: new(Options),
				out: &Options{{
					Name: &FullIdentifier{[]scanner.Token{
						make(token.Identifier, "options"),
						make(token.Identifier, "number"),
					}},
					Value: Constant{make(token.DecimalLiteral, "42")},
				}},
			},
			{name: "good syntax signed float", in: `option java_package = -10.5;`, target: new(Options),
				out: &Options{{
					Name: &FullIdentifier{[]scanner.Token{make(token.Identifier, "java_package")}},
					Value: Constant{SignedNumber{
						Sign:   make(token.Minus, ""),
						Number: make(token.FloatLiteral, "10.5"),
					}},
				}},
			},
			{name: "bad syntax", in: `option java_package = syntax;`, target: new(Options),
				err: errors.New(`expected a valid constant value, but got syntax`),
			},
		},

		"Message": {
			{name: "empty message", in: `message Foo {}`, target: new(Message),
				out: &Message{
					Name: make(token.Identifier, "Foo"),
				},
			},
			{name: "simple message", target: new(Message),
				in: `message Foo {
					repeated int32 ids = 1;
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Fields: Fields{{
							Repeated: true,
							Type:     make(token.Int32, ""),
							Name:     make(token.Identifier, "ids"),
							Number:   make(token.DecimalLiteral, "1"),
						}},
					},
				},
			},
			{name: "message with two fields", target: new(Message),
				in: `message Foo {
					bool foo = 1;
					repeated int64 ids = 2;
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Fields: Fields{
							Field{
								Type:   make(token.Bool, ""),
								Name:   make(token.Identifier, "foo"),
								Number: make(token.DecimalLiteral, "1"),
							},
							Field{
								Repeated: true,
								Type:     make(token.Int64, ""),
								Name:     make(token.Identifier, "ids"),
								Number:   make(token.DecimalLiteral, "2"),
							},
						},
					},
				},
			},
			{name: "simple message with one option", target: new(Message),
				in: `message Foo {
					repeated int32 ids = 1 [packed=true];
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Fields: Fields{{
							Repeated: true,
							Type:     make(token.Int32, ""),
							Name:     make(token.Identifier, "ids"),
							Number:   make(token.DecimalLiteral, "1"),
							Options: FieldOptions{{
								Name:  &FullIdentifier{[]scanner.Token{make(token.Identifier, "packed")}},
								Value: Constant{make(token.True, "")},
							}},
						}},
					},
				},
			},
			{name: "simple message with two options", target: new(Message),
				in: `message Foo {
					repeated int32 ids = 1 [packed=true,json="-"];
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Fields: Fields{{
							Repeated: true,
							Type:     make(token.Int32, ""),
							Name:     make(token.Identifier, "ids"),
							Number:   make(token.DecimalLiteral, "1"),
							Options: FieldOptions{{
								Name:  &FullIdentifier{[]scanner.Token{make(token.Identifier, "packed")}},
								Value: Constant{make(token.True, "")},
							}, {
								Name:  &FullIdentifier{[]scanner.Token{make(token.Identifier, "json")}},
								Value: Constant{make(token.StringLiteral, `"-"`)},
							}},
						}},
					},
				},
			},
			{name: "simple message with an enum", target: new(Message),
				in: `message Foo {
					enum EnumAllowingAlias {
						option allow_alias = true;
						UNKNOWN = 0;
						STARTED = 1;
						RUNNING = 2 [(custom_option) = "hello world"];
					}
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Enums: Enums{{
							Name: make(token.Identifier, "EnumAllowingAlias"),
							Def: EnumDef{
								Options: Options{
									{
										Name:  &FullIdentifier{[]scanner.Token{make(token.Identifier, "allow_alias")}},
										Value: Constant{make(token.True, "")},
									},
								},
								Fields: EnumFields{
									{
										Name:   make(token.Identifier, "UNKNOWN"),
										Number: make(token.DecimalLiteral, "0"),
									},
									{
										Name:   make(token.Identifier, "STARTED"),
										Number: make(token.DecimalLiteral, "1"),
									},
									{
										Name:   make(token.Identifier, "RUNNING"),
										Number: make(token.DecimalLiteral, "2"),
										Options: FieldOptions{{
											Prefix: &FullIdentifier{[]scanner.Token{make(token.Identifier, "custom_option")}},
											Value:  Constant{make(token.StringLiteral, `"hello world"`)},
										}},
									},
								},
							},
						}},
					},
				},
			},
			{name: "a message inside of a message", target: new(Message),
				in: `message Foo {
					message Bar {
					  int32 id = 1;
					}
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						Messages: Messages{{
							Name: make(token.Identifier, "Bar"),
							Def: MessageDef{
								Fields: Fields{{
									Type:   make(token.Int32, ""),
									Name:   make(token.Identifier, "id"),
									Number: make(token.DecimalLiteral, "1"),
								}},
							},
						}},
					},
				},
			},
			{name: "a message with a oneof field", target: new(Message),
				in: `message Foo {
					oneof foo {
						string name = 4;
						SubMessage sub_message = 9;
					}
				}`,
				out: &Message{
					Name: make(token.Identifier, "Foo"),
					Def: MessageDef{
						OneOfs: OneOfs{{
							Name: make(token.Identifier, "foo"),
							Fields: Fields{{
								Type:   make(token.String, ""),
								Name:   make(token.Identifier, "name"),
								Number: make(token.DecimalLiteral, "4"),
							}, {
								Type:   make(token.Identifier, "SubMessage"),
								Name:   make(token.Identifier, "sub_message"),
								Number: make(token.DecimalLiteral, "9"),
							}},
						}},
					},
				},
			},
		},
	}

	for theme, tests := range tests {
		t.Run(theme, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					log.Printf("\n\n[[[[%s]]]]\n\n", t.Name())
					s := &peeker{s: scanner.New(strings.NewReader(tt.in))}
					err := tt.target.parse(s)
					if !checkErrors(t, tt.err, err) {
						return
					}
					if !reflect.DeepEqual(tt.target, tt.out) {
						diff := pretty.Diff(tt.target, tt.out)
						// t.Fatalf("\nexpected:\n\t%v\ngot:\n\t%v\ndiff:\n\t%v", a, b, diff)
						t.Fatal(diff)
					}
				})
			}
		})
	}
}

func print(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(b)
}
