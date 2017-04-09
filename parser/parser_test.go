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

	. "github.com/campoy/groto/proto"
	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
	"github.com/kr/pretty"
)

func make(kind token.Kind, text string) scanner.Token {
	return scanner.Token{Kind: kind, Text: text}
}

func fullIdentifier(names ...string) []Identifier {
	var ids []Identifier
	for _, name := range names {
		ids = append(ids, Identifier(name))
	}
	return ids
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  *File
		err  error
	}{
		{name: "big message", in: `
			syntax = "proto3";

			package foo.bar;

			import public "new.proto";
			import "other.proto";

			option java_package = "com.example.foo";
			option go_package = "foo";

			// this is a comment

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
			out: &File{
				Syntax:  Syntax{Value: "proto3"},
				Package: Package{Identifier: fullIdentifier("foo", "bar")},
				Imports: []Import{{
					Modifier: PublicImport,
					Path:     "new.proto",
				}, {
					Path: "other.proto",
				}},
				Options: []Option{{
					Name:  fullIdentifier("java_package"),
					Value: "com.example.foo",
				}, {
					Name:  fullIdentifier("go_package"),
					Value: "foo",
				}},
				Messages: []Message{
					{
						Name: "SearchRequest",
						Fields: []Field{
							{
								Type:   Type{Predefined: TypeString},
								Name:   "query",
								Number: 1,
							}, {
								Type:   Type{Predefined: TypeInt32},
								Name:   "page_number",
								Number: 2,
							}, {
								Type:   Type{Predefined: TypeInt32},
								Name:   "result_per_page",
								Number: 3,
							}, {
								Type:   Type{UserDefined: fullIdentifier("Corpus")},
								Name:   "corpus",
								Number: 4,
							},
						},
						Enums: []Enum{
							{
								Name: "Corpus",
								Fields: []EnumField{
									{Name: "UNIVERSAL", Number: 0},
									{Name: "WEB", Number: 1},
									{Name: "IMAGES", Number: 2},
									{Name: "LOCAL", Number: 3},
									{Name: "NEWS", Number: 4},
									{Name: "PRODUCTS", Number: 5},
									{Name: "VIDEO", Number: 6},
								},
							},
						},
					}, {
						Name: "Foo",
						Reserveds: []Reserved{
							{
								IDs:    []int{2, 15},
								Ranges: []Range{{From: 9, To: 11}},
							}, {
								Names: []string{"foo", "bar"},
							},
						},
					},
				},
				Enums: []Enum{
					{
						Name: "EnumAllowingAlias",
						Fields: []EnumField{
							{Name: "UNKNOWN", Number: 0},
							{Name: "STARTED", Number: 1},
							{Name: "RUNNING", Number: 1},
						},
						Options: []Option{
							{Name: fullIdentifier("allow_alias"), Value: true},
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
			out: Syntax{Value: "proto3"},
		},
		{name: "missing equal", in: `syntax "proto3";`,
			err: errors.New(`expected '=', got string literal ("proto3")`),
		},
		{name: "bad text", in: `syntax = "proto2";`,
			err: errors.New(`expected literal string proto3, got proto2 instead`),
		},
		{name: "missing semicolon", in: `syntax = "proto3"`,
			err: errors.New(`expected ';', got end of file`),
		},
		{name: "missing quotes", in: `syntax = proto3;`,
			err: errors.New(`expected string literal, got identifier (proto3)`),
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
			out: Import{Path: "path"},
		},
		{name: "good public syntax", in: `import public "path";`,
			out: Import{Path: "path", Modifier: PublicImport},
		},
		{name: "good weak syntax", in: `import weak "path";`,
			out: Import{Path: "path", Modifier: WeakImport},
		},
		{name: "bad modifier", in: `import bytes "path";`,
			err: errors.New(`expected string literal, got bytes`),
		},
		{name: "bad modifier keyword", in: `import enum "path";`,
			err: errors.New(`expected string literal, got enum`),
		},
		{name: "bad import path", in: `import public path;`,
			err: errors.New(`expected string literal, got identifier (path)`),
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
			out: Package{Identifier: fullIdentifier("foo")},
		},
		{name: "full identifer", in: `package com.example.foo;`,
			out: Package{Identifier: fullIdentifier("com", "example", "foo")},
		},
		{name: "bad identifier", in: `package "foo";`,
			err: errors.New(`expected identifier, got string literal ("foo")`),
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
				Name:  fullIdentifier("java_package"),
				Value: "com.example.foo",
			},
		},
		{name: "good syntax full identifer", in: `option java_package = foo.bar;`,
			out: Option{
				Name:  fullIdentifier("java_package"),
				Value: fullIdentifier("foo", "bar"),
			},
		},
		{name: "good syntax integer", in: `option options.number = 42;`,
			out: Option{
				Name:  fullIdentifier("options", "number"),
				Value: int64(42),
			},
		},
		{name: "good syntax signed float", in: `option java_package = -10.5;`,
			out: Option{
				Name:  fullIdentifier("java_package"),
				Value: -10.5,
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
			out: Message{Name: "Foo"},
		},
		{name: "simple message",
			in: `message Foo {
					repeated int32 ids = 1;
				}`,
			out: Message{
				Name: "Foo",
				Fields: []Field{{
					Repeated: true,
					Type:     Type{Predefined: TypeInt32},
					Name:     "ids",
					Number:   1,
				}},
			},
		},
		{name: "message with two fields",
			in: `message Foo {
					bool foo = 1;
					repeated int64 ids = 2;
				}`,
			out: Message{
				Name: "Foo",
				Fields: []Field{{
					Type:   Type{Predefined: TypeBool},
					Name:   "foo",
					Number: 1,
				}, {
					Repeated: true,
					Type:     Type{Predefined: TypeInt64},
					Name:     "ids",
					Number:   2,
				}},
			},
		},
		{name: "simple message with one option",
			in: `message Foo {
					repeated int32 ids = 1 [packed=true];
				}`,
			out: Message{
				Name: "Foo",
				Fields: []Field{{
					Repeated: true,
					Type:     Type{Predefined: TypeInt32},
					Name:     "ids",
					Number:   1,
					Options: []Option{{
						Name:  fullIdentifier("packed"),
						Value: true,
					}},
				}},
			},
		},
		{name: "simple message with two options",
			in: `message Foo {
					repeated int32 ids = 1 [packed=true,json="-"];
				}`,
			out: Message{
				Name: "Foo",
				Fields: []Field{{
					Repeated: true,
					Type:     Type{Predefined: TypeInt32},
					Name:     "ids",
					Number:   1,
					Options: []Option{{
						Name:  fullIdentifier("packed"),
						Value: true,
					}, {
						Name:  fullIdentifier("json"),
						Value: "-",
					}},
				}},
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
				Name: "Foo",
				Enums: []Enum{{
					Name: "EnumAllowingAlias",
					Options: []Option{
						{
							Name:  fullIdentifier("allow_alias"),
							Value: true,
						},
					},
					Fields: []EnumField{
						{Name: "UNKNOWN", Number: 0},
						{Name: "STARTED", Number: 1},
						{
							Name:   "RUNNING",
							Number: 2,
							Options: []Option{{
								Prefix: fullIdentifier("custom_option"),
								Value:  "hello world",
							}},
						},
					},
				}},
			},
		},
		{name: "a message inside of a message",
			in: `message Foo {
					message Bar {
					  int32 id = 1;
					}
				}`,
			out: Message{
				Name: "Foo",
				Messages: []Message{{
					Name: "Bar",
					Fields: []Field{{
						Type:   Type{Predefined: TypeInt32},
						Name:   "id",
						Number: 1,
					}},
				}},
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
				Name: "Foo",
				OneOfs: []OneOf{{
					Name: "foo",
					Fields: []OneOfField{{
						Type:   Type{Predefined: TypeString},
						Name:   "name",
						Number: 4,
					}, {
						Type:   Type{UserDefined: fullIdentifier("SubMessage")},
						Name:   "sub_message",
						Number: 9,
					}},
				}},
			},
		},
		{name: "a message with a map field",
			in: `message Foo {
					map<string, Project> projects = 3;
				}`,
			out: Message{
				Name: "Foo",
				Maps: []Map{{
					Name:      "projects",
					KeyType:   Type{Predefined: TypeString},
					ValueType: Type{UserDefined: fullIdentifier("Project")},
					Number:    3,
				}},
			},
		},
		{name: "a message with a map field",
			in: `message Foo {
					reserved 2, 15, 9 to 11;
					reserved "foo", "bar";
				}`,
			out: Message{
				Name: "Foo",
				Reserveds: []Reserved{{
					IDs:    []int{2, 15},
					Ranges: []Range{{From: 9, To: 11}},
				}, {
					Names: []string{"foo", "bar"},
				}},
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
				Name: "SearchService",
				RPCs: []RPC{{
					Name: "Search",
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
				Name: "SearchService",
				RPCs: []RPC{{
					Name: "Search",
					In:   RPCParam{Type: fullIdentifier("SearchRequest")},
					Out:  RPCParam{Type: fullIdentifier("SearchResponse"), Stream: true},
					Options: []Option{{
						Name:  fullIdentifier("secured"),
						Value: false,
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
