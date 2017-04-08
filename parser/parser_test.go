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

package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
	"github.com/kr/pretty"
)

func make(kind token.Kind, text string) scanner.Token {
	return scanner.Token{Kind: kind, Text: text}
}

func fullIdentifier(names ...string) FullIdentifier {
	var f FullIdentifier
	for _, name := range names {
		f.Identifiers = append(f.Identifiers, make(token.Identifier, name))
	}
	return f
}

func ptrFullIdentifier(names ...string) *FullIdentifier {
	f := fullIdentifier(names...)
	return &f
}

func TestParseProto(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *Proto
		err  error
	}{
		{name: "big message", in: `
			syntax = "proto3";

			package foo.bar;

			import public "new.proto";
			import "other.proto";

			option java_package = "com.example.foo";
			option go_package = "foo";

			message SearchRequest {
				string query = 1;
				int32 page_number = 2;  // Which page number do we want?
				int32 result_per_page = 3;  // Number of results to return per page.
				enum Corpus {
					UNIVERSAL = 0;
					WEB = 1;
					IMAGES = 2;
					LOCAL = 3;
					NEWS = 4;
					PRODUCTS = 5;
					VIDEO = 6;
				}
				Corpus corpus = 4;
			}

			message Foo {
				reserved 2, 15, 9 to 11;
				reserved "foo", "bar";
			}

			enum EnumAllowingAlias {
				option allow_alias = true;
				UNKNOWN = 0;
				STARTED = 1;
				RUNNING = 1;
			}
			`,
			out: &Proto{
				Syntax:  Syntax{make(token.StringLiteral, `"proto3"`)},
				Package: Package{fullIdentifier("foo", "bar")},
				Imports: []Import{{
					Modifier: make(token.Public, ""),
					Path:     make(token.StringLiteral, `"new.proto"`),
				}, {
					Path: make(token.StringLiteral, `"other.proto"`),
				}},
				Options: []Option{{
					Name:  ptrFullIdentifier("java_package"),
					Value: Constant{make(token.StringLiteral, `"com.example.foo"`)},
				}, {
					Name:  ptrFullIdentifier("go_package"),
					Value: Constant{make(token.StringLiteral, `"foo"`)},
				}},
				Messages: []Message{
					{
						Name: make(token.Identifier, "SearchRequest"),
						Def: MessageDef{
							Fields: []Field{
								{
									Type:   make(token.String, ""),
									Name:   make(token.Identifier, "query"),
									Number: make(token.DecimalLiteral, "1"),
								}, {
									Type:   make(token.Int32, ""),
									Name:   make(token.Identifier, "page_number"),
									Number: make(token.DecimalLiteral, "2"),
								}, {
									Type:   make(token.Int32, ""),
									Name:   make(token.Identifier, "result_per_page"),
									Number: make(token.DecimalLiteral, "3"),
								}, {
									Type:   make(token.Identifier, "Corpus"),
									Name:   make(token.Identifier, "corpus"),
									Number: make(token.DecimalLiteral, "4"),
								},
							},
							Enums: []Enum{
								{
									Name: make(token.Identifier, "Corpus"),
									Def: EnumDef{
										Fields: []EnumField{
											{
												Name:   make(token.Identifier, "UNIVERSAL"),
												Number: make(token.DecimalLiteral, "0"),
											}, {
												Name:   make(token.Identifier, "WEB"),
												Number: make(token.DecimalLiteral, "1"),
											}, {
												Name:   make(token.Identifier, "IMAGES"),
												Number: make(token.DecimalLiteral, "2"),
											}, {
												Name:   make(token.Identifier, "LOCAL"),
												Number: make(token.DecimalLiteral, "3"),
											}, {
												Name:   make(token.Identifier, "NEWS"),
												Number: make(token.DecimalLiteral, "4"),
											}, {
												Name:   make(token.Identifier, "PRODUCTS"),
												Number: make(token.DecimalLiteral, "5"),
											}, {
												Name:   make(token.Identifier, "VIDEO"),
												Number: make(token.DecimalLiteral, "6"),
											},
										},
									},
								},
							},
						},
					}, {
						Name: make(token.Identifier, "Foo"),
						Def: MessageDef{
							Reserveds: []Reserved{
								{
									IDs: []scanner.Token{
										make(token.DecimalLiteral, "2"),
										make(token.DecimalLiteral, "15"),
									},
									Ranges: []Range{{
										make(token.DecimalLiteral, "9"),
										make(token.DecimalLiteral, "11"),
									}},
								}, {
									Names: []scanner.Token{
										make(token.StringLiteral, `"foo"`),
										make(token.StringLiteral, `"bar"`),
									},
								},
							},
						},
					},
				},
				Enums: []Enum{
					{
						Name: make(token.Identifier, "EnumAllowingAlias"),
						Def: EnumDef{
							Fields: []EnumField{
								{
									Name:   make(token.Identifier, "UNKNOWN"),
									Number: make(token.DecimalLiteral, "0"),
								}, {
									Name:   make(token.Identifier, "STARTED"),
									Number: make(token.DecimalLiteral, "1"),
								}, {
									Name:   make(token.Identifier, "RUNNING"),
									Number: make(token.DecimalLiteral, "1"),
								},
							},
							Options: []Option{
								{
									Name:  ptrFullIdentifier("allow_alias"),
									Value: Constant{make(token.True, "")},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			proto, err := parseProto(p)
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, proto)
		})
	}
}

func TestParseSyntax(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Syntax
		err  error
	}{
		{name: "good syntax", in: `syntax = "proto3";`,
			out: Syntax{make(token.StringLiteral, `"proto3"`)},
		},
		{name: "missing equal", in: `syntax "proto3";`,
			err: errors.New(`expected Equals, got string literal ("proto3")`),
		},
		{name: "bad text", in: `syntax = "proto2";`,
			err: errors.New(`expected literal string "proto3", got "proto2" instead`),
		},
		{name: "missing semicolon", in: `syntax = "proto3"`,
			err: errors.New(`expected Semicolon, got e o f`),
		},
		{name: "missing quotes", in: `syntax = proto3;`,
			err: errors.New(`expected StringLiteral, got identifier (proto3)`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var syntax Syntax
			err := panicToErr(func() { syntax = parseSyntax(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, syntax)
		})
	}
}

func TestParseImport(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Import
		err  error
	}{
		{name: "good syntax", in: `import "path";`,
			out: Import{Path: make(token.StringLiteral, `"path"`)},
		},
		{name: "good public syntax", in: `import public "path";`,
			out: Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Public, "")},
		},
		{name: "good weak syntax", in: `import weak "path";`,
			out: Import{Path: make(token.StringLiteral, `"path"`), Modifier: make(token.Weak, "")},
		},
		{name: "bad modifier", in: `import bytes "path";`,
			err: errors.New(`expected StringLiteral, got bytes`),
		},
		{name: "bad modifier keyword", in: `import enum "path";`,
			err: errors.New(`expected StringLiteral, got enum`),
		},
		{name: "bad import path", in: `import public path;`,
			err: errors.New(`expected StringLiteral, got identifier (path)`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var imp Import
			err := panicToErr(func() { imp = parseImport(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, imp)
		})
	}
}

func TestParsePackage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Package
		err  error
	}{
		{name: "single identifier", in: `package foo;`,
			out: Package{fullIdentifier("foo")},
		},
		{name: "full identifer", in: `package com.example.foo;`,
			out: Package{fullIdentifier("com", "example", "foo")},
		},
		{name: "bad identifier", in: `package "foo";`,
			err: errors.New(`expected Identifier, got string literal ("foo")`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var pkg Package
			err := panicToErr(func() { pkg = parsePackage(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, pkg)
		})
	}
}

func TestParseOption(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Option
		err  error
	}{
		{name: "good syntax string", in: `option java_package = "com.example.foo";`,
			out: Option{
				Name:  ptrFullIdentifier("java_package"),
				Value: Constant{make(token.StringLiteral, `"com.example.foo"`)},
			},
		},
		{name: "good syntax full identifer", in: `option java_package = foo.bar;`,
			out: Option{
				Name:  ptrFullIdentifier("java_package"),
				Value: Constant{fullIdentifier("foo", "bar")},
			},
		},
		{name: "good syntax integer", in: `option options.number = 42;`,
			out: Option{
				Name:  ptrFullIdentifier("options", "number"),
				Value: Constant{make(token.DecimalLiteral, "42")},
			},
		},
		{name: "good syntax signed float", in: `option java_package = -10.5;`,
			out: Option{
				Name: ptrFullIdentifier("java_package"),
				Value: Constant{SignedNumber{
					Sign:   make(token.Minus, ""),
					Number: make(token.FloatLiteral, "10.5"),
				}},
			},
		},
		{name: "bad syntax", in: `option java_package = syntax;`,
			err: errors.New(`expected a valid constant value, but got syntax`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var opt Option
			err := panicToErr(func() { opt = parseOption(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, opt)
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Message
		err  error
	}{
		{name: "empty message", in: `message Foo {}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
			},
		},
		{name: "simple message",
			in: `message Foo {
					repeated int32 ids = 1;
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Fields: []Field{{
						Repeated: true,
						Type:     make(token.Int32, ""),
						Name:     make(token.Identifier, "ids"),
						Number:   make(token.DecimalLiteral, "1"),
					}},
				},
			},
		},
		{name: "message with two fields",
			in: `message Foo {
					bool foo = 1;
					repeated int64 ids = 2;
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Fields: []Field{{
						Type:   make(token.Bool, ""),
						Name:   make(token.Identifier, "foo"),
						Number: make(token.DecimalLiteral, "1"),
					}, {
						Repeated: true,
						Type:     make(token.Int64, ""),
						Name:     make(token.Identifier, "ids"),
						Number:   make(token.DecimalLiteral, "2"),
					},
					},
				},
			},
		},
		{name: "simple message with one option",
			in: `message Foo {
					repeated int32 ids = 1 [packed=true];
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Fields: []Field{{
						Repeated: true,
						Type:     make(token.Int32, ""),
						Name:     make(token.Identifier, "ids"),
						Number:   make(token.DecimalLiteral, "1"),
						Options: []Option{{
							Name:  ptrFullIdentifier("packed"),
							Value: Constant{make(token.True, "")},
						}},
					}},
				},
			},
		},
		{name: "simple message with two options",
			in: `message Foo {
					repeated int32 ids = 1 [packed=true,json="-"];
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Fields: []Field{{
						Repeated: true,
						Type:     make(token.Int32, ""),
						Name:     make(token.Identifier, "ids"),
						Number:   make(token.DecimalLiteral, "1"),
						Options: []Option{{
							Name:  ptrFullIdentifier("packed"),
							Value: Constant{make(token.True, "")},
						}, {
							Name:  ptrFullIdentifier("json"),
							Value: Constant{make(token.StringLiteral, `"-"`)},
						}},
					}},
				},
			},
		},
		{name: "simple message with an enum",
			in: `message Foo {
					enum EnumAllowingAlias {
						option allow_alias = true;
						UNKNOWN = 0;
						STARTED = 1;
						RUNNING = 2 [(custom_option) = "hello world"];
					}
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Enums: []Enum{{
						Name: make(token.Identifier, "EnumAllowingAlias"),
						Def: EnumDef{
							Options: []Option{
								{
									Name:  ptrFullIdentifier("allow_alias"),
									Value: Constant{make(token.True, "")},
								},
							},
							Fields: []EnumField{{
								Name:   make(token.Identifier, "UNKNOWN"),
								Number: make(token.DecimalLiteral, "0"),
							}, {
								Name:   make(token.Identifier, "STARTED"),
								Number: make(token.DecimalLiteral, "1"),
							}, {
								Name:   make(token.Identifier, "RUNNING"),
								Number: make(token.DecimalLiteral, "2"),
								Options: []Option{{
									Prefix: ptrFullIdentifier("custom_option"),
									Value:  Constant{make(token.StringLiteral, `"hello world"`)},
								}},
							},
							},
						},
					}},
				},
			},
		},
		{name: "a message inside of a message",
			in: `message Foo {
					message Bar {
					  int32 id = 1;
					}
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Messages: []Message{{
						Name: make(token.Identifier, "Bar"),
						Def: MessageDef{
							Fields: []Field{{
								Type:   make(token.Int32, ""),
								Name:   make(token.Identifier, "id"),
								Number: make(token.DecimalLiteral, "1"),
							}},
						},
					}},
				},
			},
		},
		{name: "a message with a oneof field",
			in: `message Foo {
					oneof foo {
						string name = 4;
						SubMessage sub_message = 9;
					}
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					OneOfs: []OneOf{{
						Name: make(token.Identifier, "foo"),
						Fields: []Field{{
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
		{name: "a message with a map field",
			in: `message Foo {
					map<string, Project> projects = 3;
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Maps: []Map{{
						Name:      make(token.Identifier, "projects"),
						KeyType:   make(token.String, ""),
						ValueType: fullIdentifier("Project"),
						Number:    make(token.DecimalLiteral, "3"),
					}},
				},
			},
		},
		{name: "a message with a map field",
			in: `message Foo {
					reserved 2, 15, 9 to 11;
					reserved "foo", "bar";
				}`,
			out: Message{
				Name: make(token.Identifier, "Foo"),
				Def: MessageDef{
					Reserveds: []Reserved{{
						IDs: []scanner.Token{
							make(token.DecimalLiteral, "2"),
							make(token.DecimalLiteral, "15"),
						},
						Ranges: []Range{{
							make(token.DecimalLiteral, "9"),
							make(token.DecimalLiteral, "11"),
						}},
					}, {
						Names: []scanner.Token{
							make(token.StringLiteral, `"foo"`),
							make(token.StringLiteral, `"bar"`),
						},
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var msg Message
			err := panicToErr(func() { msg = parseMessage(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, msg)
		})
	}
}

func TestParseService(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  Service
		err  error
	}{
		{name: "good syntax string", in: `
				service SearchService {
					rpc Search (SearchRequest) returns (stream SearchResponse);
				}`,
			out: Service{
				Name: make(token.Identifier, "SearchService"),
				RPCs: []RPC{{
					Name: make(token.Identifier, "Search"),
					In:   RPCParam{Type: fullIdentifier("SearchRequest")},
					Out:  RPCParam{Type: fullIdentifier("SearchResponse"), Stream: true},
				}},
			},
		},
		{name: "good syntax string", in: `
				service SearchService {
					rpc Search (SearchRequest) returns (stream SearchResponse) {
						option secured = false;
					}
				}`,
			out: Service{
				Name: make(token.Identifier, "SearchService"),
				RPCs: []RPC{{
					Name: make(token.Identifier, "Search"),
					In:   RPCParam{Type: fullIdentifier("SearchRequest")},
					Out:  RPCParam{Type: fullIdentifier("SearchResponse"), Stream: true},
					Options: []Option{{
						Name:  ptrFullIdentifier("secured"),
						Value: Constant{make(token.False, "")},
					}},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &peeker{s: scanner.New(strings.NewReader(tt.in))}
			var svc Service
			err := panicToErr(func() { svc = parseService(p) })
			if !checkErrors(t, tt.err, err) {
				return
			}
			checkResults(t, tt.out, svc)
		})
	}
}

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

func checkResults(t *testing.T, want, got interface{}) {
	if !reflect.DeepEqual(want, got) {
		diff := pretty.Diff(want, got)
		log.Printf("expected: %s", print(want))
		log.Printf("got: %s", print(got))
		t.Fatal(diff)
	}
}

func panicToErr(f func()) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", rec)
		}
	}()
	f()
	return nil
}

func print(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(b)
}
