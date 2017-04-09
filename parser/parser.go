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

// Package parser provides the function Parse, which given an io.Reader parses
// its content and generates a Proto which contains all the definitions found
// in a Protocol Buffer Version 3 file descriptor (aka .proto file).
//
// You can read more about the language here:
// https://developers.google.com/protocol-buffers/docs/proto3#oneof
package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	. "github.com/campoy/groto/proto"
	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

// Parse reads from the given io.Reader and returns the parsed information in
// a Proto value, or an error if the contents where not parseable.
func Parse(r io.Reader) (*File, error) {
	return parseProto(&peeker{s: scanner.New(r)})
}

// proto = syntax { import | package | option |  message | enum | service | emptyStatement }
func parseProto(p *peeker) (file *File, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", rec)
		}
	}()

	file = &File{Syntax: parseSyntax(p)}
	for {
		switch next := p.peek(); next.Kind {
		case token.Package:
			if len(file.Package.Identifier) > 0 {
				panicf("found second package definition")
			}
			file.Package = parsePackage(p)
		case token.Import:
			file.Imports = append(file.Imports, parseImport(p))
		case token.Option:
			file.Options = append(file.Options, parseOption(p))
		case token.Message:
			file.Messages = append(file.Messages, parseMessage(p))
		case token.Enum:
			file.Enums = append(file.Enums, parseEnum(p))
		case token.Service:
			file.Services = append(file.Services, parseService(p))
		case token.EOF:
			return file, nil
		default:
			panicf("unexpected %s at top level definition", next)
		}
	}
}

// syntax = "syntax" "=" quote "proto3" quote ";"
func parseSyntax(p *peeker) Syntax {
	p.consume(token.Syntax)
	p.consume(token.Equals)
	value := unquote(p.consume(token.StringLiteral))
	if value != "proto3" {
		panicf("expected literal string proto3, got %s instead", value)
	}
	p.consume(token.Semicolon)
	return Syntax{Value: value}
}

// import = "import" [ "weak" | "public" ] strLit ";"
func parseImport(p *peeker) Import {
	p.consume(token.Import)
	var mod ImportModifier
	if tok, ok := p.maybeConsume(token.Weak, token.Public); ok {
		if tok.Is(token.Weak) {
			mod = WeakImport
		} else {
			mod = PublicImport
		}
	}
	path := unquote(p.consume(token.StringLiteral))
	p.consume(token.Semicolon)
	return Import{Modifier: mod, Path: path}
}

// package = "package" fullIdent ";"
func parsePackage(p *peeker) Package {
	p.consume(token.Package)
	ident := parseFullIdentifier(p)
	p.consume(token.Semicolon)
	return Package{Identifier: ident}
}

// option = "option" optionName  "=" constant ";"
// optionName = ( ident | "(" fullIdent ")" ) { "." ident }
func parseOption(p *peeker) Option {
	p.consume(token.Option)
	opt := parseFieldOption(p)
	p.consume(token.Semicolon)
	return opt
}

// fieldOption = optionName "=" constant
func parseFieldOption(p *peeker) Option {
	var opt Option
	if _, ok := p.maybeConsume(token.OpenParen); ok {
		opt.Prefix = parseFullIdentifier(p)
		p.consume(token.CloseParen)
	}

	if p.peek().Is(token.Identifier) {
		opt.Name = parseFullIdentifier(p)
	}

	if opt.Prefix == nil && opt.Name == nil {
		panicf("missing name in option")
	}

	p.consume(token.Equals)
	opt.Value = parseValue(p, false)
	return opt
}

// fieldOptions = [ "[" fieldOption { ","  fieldOption } "]" ]
func parseFieldOptions(p *peeker) []Option {
	if _, ok := p.maybeConsume(token.OpenBracket); !ok {
		return nil
	}

	var opts []Option
	for {
		opts = append(opts, parseFieldOption(p))
		if _, ok := p.maybeConsume(token.CloseBracket); ok {
			return opts
		}
		p.consume(token.Comma)
	}
}

// message = "message" messageName messageBody
// messageBody = "{" { field | enum | message | option | oneof | mapField | reserved | emptyStatement } "}"
func parseMessage(p *peeker) Message {
	p.consume(token.Message)
	msg := Message{Name: identifier(p.consume(token.Identifier))}
	p.consume(token.OpenBrace)

	for {
		switch kind := p.peek().Kind; {
		case kind.IsType() || kind == token.Identifier || kind == token.Repeated:
			msg.Fields = append(msg.Fields, parseField(p))
		case kind == token.Enum:
			msg.Enums = append(msg.Enums, parseEnum(p))
		case kind == token.Message:
			msg.Messages = append(msg.Messages, parseMessage(p))
		case kind == token.Option:
			msg.Options = append(msg.Options, parseOption(p))
		case kind == token.Oneof:
			msg.OneOfs = append(msg.OneOfs, parseOneOf(p))
		case kind == token.Map:
			msg.Maps = append(msg.Maps, parseMap(p))
		case kind == token.Reserved:
			msg.Reserveds = append(msg.Reserveds, parseReserved(p))
		case kind == token.Semicolon:
			p.scan()
		case kind == token.CloseBrace:
			p.scan()
			return msg
		default:
			panicf("expected '}' to end message definition, got %s", p.scan())
		}
	}
}

// field = [ "repeated" ] type fieldName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseField(p *peeker) Field {
	_, repeated := p.maybeConsume(token.Repeated)
	f := parseOneOfField(p)
	return Field{Repeated: repeated, Type: f.Type, Name: f.Name, Number: f.Number, Options: f.Options}
}

// enum = "enum" enumName "{" { option | enumField | emptyStatement } "}"
func parseEnum(p *peeker) Enum {
	p.consume(token.Enum)
	enum := Enum{Name: identifier(p.consume(token.Identifier))}
	p.consume(token.OpenBrace)

	for {
		switch kind := p.peek().Kind; {
		case kind.IsType() || kind == token.Identifier || kind == token.Repeated:
			enum.Fields = append(enum.Fields, parseEnumField(p))
		case kind == token.Option:
			enum.Options = append(enum.Options, parseOption(p))
		case kind == token.CloseBrace:
			p.scan()
			return enum
		default:
			panicf("expected '}' to end message definition, got %s", p.scan())
		}
	}
}

// enumField = ident "=" intLit fieldOptions ";"
func parseEnumField(p *peeker) EnumField {
	name := identifier(p.consume(token.Identifier))
	p.consume(token.Equals)
	number := atoi(p.consume(token.DecimalLiteral))
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return EnumField{Name: name, Number: number, Options: opts}
}

// oneof = "oneof" oneofName "{" { oneofField | emptyStatement } "}"
func parseOneOf(p *peeker) OneOf {
	p.consume(token.Oneof)
	o := OneOf{Name: identifier(p.consume(token.Identifier))}
	p.consume(token.OpenBrace)

	for {
		if _, ok := p.maybeConsume(token.CloseBrace); ok {
			return o
		}
		o.Fields = append(o.Fields, parseOneOfField(p))
	}
}

// oneofField = type fieldName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseOneOfField(p *peeker) OneOfField {
	typ := parseType(p)
	name := identifier(p.consume(token.Identifier))
	p.consume(token.Equals)
	number := atoi(p.consume(token.DecimalLiteral))
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return OneOfField{Type: typ, Name: name, Number: number, Options: opts}
}

// mapField = "map" "<" keyType "," type ">" mapName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseMap(p *peeker) Map {
	p.consume(token.Map)
	p.consume(token.OpenAngled)
	key := p.scan()
	if !key.IsKeyType() {
		panicf("expected key type, got %s", key)
	}
	keyType := Type{Predefined: kindToType(key.Kind)}
	p.consume(token.Comma)
	valueType := parseType(p)
	p.consume(token.CloseAngled)
	name := identifier(p.consume(token.Identifier))
	p.consume(token.Equals)
	number := atoi(p.consume(token.DecimalLiteral))
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return Map{KeyType: keyType, ValueType: valueType, Name: name, Number: number, Options: opts}
}

// type = "double" | "float" | "int32" | "int64" | "uint32" | "uint64"
//       | "sint32" | "sint64" | "fixed32" | "fixed64" | "sfixed32" | "sfixed64"
//       | "bool" | "string" | "bytes" | messageType | enumType
func parseType(p *peeker) Type {
	if p.peek().IsType() {
		return Type{Predefined: kindToType(p.scan().Kind)}
	}
	return Type{UserDefined: parseFullIdentifier(p)}
}

// reserved = "reserved" ( ranges | fieldNames ) ";"
// fieldNames = fieldName { "," fieldName }
func parseReserved(p *peeker) Reserved {
	p.consume(token.Reserved)

	var res Reserved
	for {
		from, ok := p.maybeConsume(token.DecimalLiteral, token.StringLiteral)
		if !ok {
			panicf("expected decimal literal or string, got %s", from)
		}
		if _, ok := p.maybeConsume(token.To); ok {
			to := p.consume(from.Kind)
			if !from.Is(token.DecimalLiteral) {
				panicf("ranges over strings are not supported %s to %s", from, to)
			}
			res.Ranges = append(res.Ranges, Range{From: atoi(from), To: atoi(to)})
		} else {
			if from.Is(token.DecimalLiteral) {
				res.IDs = append(res.IDs, atoi(from))
			} else {
				res.Names = append(res.Names, unquote(from))
			}
		}
		if _, ok := p.maybeConsume(token.Semicolon); ok {
			return res
		}
		p.consume(token.Comma)
	}
}

// service = "service" serviceName "{" { option | rpc | emptyStatement } "}"
func parseService(p *peeker) Service {
	p.consume(token.Service)
	svc := Service{Name: identifier(p.consume(token.Identifier))}

	p.consume(token.OpenBrace)
	for {
		switch p.peek().Kind {
		case token.Option:
			svc.Options = append(svc.Options, parseOption(p))
		case token.RPC:
			svc.RPCs = append(svc.RPCs, parseRPC(p))
		case token.CloseBrace:
			p.scan()
			return svc
		default:
			panicf("expected option or rpc in service, got %s", p.peek())
		}
	}
}

// rpc = "rpc" rpcName rpcParam "returns" rpcParam (( "{" {option | emptyStatement } "}" ) | ";")
func parseRPC(p *peeker) RPC {
	p.consume(token.RPC)
	rpc := RPC{Name: identifier(p.consume(token.Identifier))}
	rpc.In = parseRPCParam(p)
	p.consume(token.Returns)
	rpc.Out = parseRPCParam(p)

	if _, ok := p.maybeConsume(token.Semicolon); ok {
		return rpc
	}

	p.consume(token.OpenBrace)
	for {
		if _, ok := p.maybeConsume(token.CloseBrace); ok {
			return rpc
		}
		rpc.Options = append(rpc.Options, parseOption(p))
	}
}

// rpcParam = "(" [ "stream" ] messageType ")"
func parseRPCParam(p *peeker) RPCParam {
	p.consume(token.OpenParen)
	_, stream := p.maybeConsume(token.Stream)
	typ := parseFullIdentifier(p)
	p.consume(token.CloseParen)
	return RPCParam{Stream: stream, Type: typ}
}

func identifier(tok scanner.Token) Identifier {
	if !tok.Is(token.Identifier) {
		panicf("can't parse an identifier from %s", tok)
	}
	return Identifier(tok.Text)
}

func parseFullIdentifier(p *peeker) []Identifier {
	ident := []Identifier{identifier(p.consume(token.Identifier))}
	for {
		dot := p.peek()
		if !dot.Is(token.Dot) {
			return ident
		}
		p.scan()
		ident = append(ident, identifier(p.consume(token.Identifier)))
	}
}

func parseInt(tok scanner.Token, base int, negative bool) int64 {
	v, err := strconv.ParseInt(tok.Text, base, 64)
	if err != nil {
		panicf("bad %s: %v", tok, err)
	}
	if negative {
		return -v
	}
	return v
}

func parseFloat(tok scanner.Token, negative bool) float64 {
	v, err := strconv.ParseFloat(tok.Text, 64)
	if err != nil {
		panicf("bad %s: %v", tok, err)
	}
	if negative {
		return -v
	}
	return v
}

func parseValue(p *peeker, negative bool) interface{} {
	next := p.peek()
	if next.Is(token.Identifier) {
		return parseFullIdentifier(p)
	}
	p.scan()
	switch next.Kind {
	case token.DecimalLiteral:
		return parseInt(next, 10, negative)
	case token.HexLiteral:
		return parseInt(next, 16, negative)
	case token.OctalLiteral:
		return parseInt(next, 8, negative)
	case token.FloatLiteral:
		return parseFloat(next, negative)
	default:
		if negative {
			panicf("found minus sign before %s", next)
		}
		switch next.Kind {
		case token.StringLiteral:
			return unquote(next)
		case token.False:
			return false
		case token.True:
			return true
		case token.Minus:
			return parseValue(p, true)
		case token.Plus:
			return parseValue(p, false)
		default:
			panicf("expected a valid constant value, but got %s", next)
		}
	}
	panic("unreachable")
}

func panicf(format string, args ...interface{}) { panic(fmt.Sprintf(format, args...)) }

type peeker struct {
	s      *scanner.Scanner
	peeked *scanner.Token
}

func (p *peeker) scan() (res scanner.Token) {
	if tok := p.peeked; tok != nil {
		p.peeked = nil
		return *tok
	}
	tok := p.s.Scan()
	for tok.Is(token.Comment) {
		tok = p.s.Scan()
	}
	return tok
}

func (p *peeker) peek() (res scanner.Token) {
	if tok := p.peeked; tok != nil {
		return *tok
	}
	tok := p.s.Scan()
	for tok.Is(token.Comment) {
		tok = p.s.Scan()
	}
	p.peeked = &tok
	return tok
}

// consumes and returns a token of one of the given kinds or panics
func (p *peeker) consume(toks ...token.Kind) scanner.Token {
	got := p.scan()
	for _, tok := range toks {
		if got.Kind == tok {
			return got
		}
	}
	var types []string
	for _, tok := range toks {
		types = append(types, fmt.Sprint(tok))
	}
	panicf("expected %v, got %s", strings.Join(types, ", "), got)
	panic("unreachable")
}

// consumes and returns a token of one of the given kinds or returns what it found and false.
func (p *peeker) maybeConsume(toks ...token.Kind) (scanner.Token, bool) {
	got := p.peek()
	for _, tok := range toks {
		if got.Kind == tok {
			p.scan()
			return got, true
		}
	}
	return got, false
}

func atoi(tok scanner.Token) int {
	if !tok.IsNumber() {
		panicf("can't parse a number from %s", tok)
	}
	v, err := strconv.Atoi(tok.Text)
	if err != nil {
		panicf("bad number %s", tok.Text)
	}
	return v
}

func unquote(tok scanner.Token) string {
	if !tok.Is(token.StringLiteral) {
		panicf("can't unquote %s", tok)
	}
	v, err := strconv.Unquote(tok.Text)
	if err != nil {
		panicf("bad quoting on %s: %v", tok, err)
	}
	return v
}

func kindToType(k token.Kind) PredefinedType {
	t, ok := kindToTypeMap[k]
	if !ok {
		panicf("could not find predefined type for %s", k)
	}
	return t
}

var kindToTypeMap = map[token.Kind]PredefinedType{
	token.Bytes:    TypeBytes,
	token.Double:   TypeDouble,
	token.Float:    TypeFloat,
	token.Bool:     TypeBool,
	token.Fixed32:  TypeFixed32,
	token.Fixed64:  TypeFixed64,
	token.Int32:    TypeInt32,
	token.Int64:    TypeInt64,
	token.Sfixed32: TypeSfixed32,
	token.Sfixed64: TypeSfixed64,
	token.Sint32:   TypeSint32,
	token.Sint64:   TypeSint64,
	token.String:   TypeString,
	token.Uint32:   TypeUint32,
	token.Uint64:   TypeUint64,
}
