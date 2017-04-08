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
	"fmt"
	"io"
	"strings"

	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

type Proto struct {
	Syntax   Syntax
	Package  Package
	Imports  []Import
	Options  []Option
	Messages []Message
	Enums    []Enum
	Services []Service
}

func Parse(r io.Reader) (*Proto, error) {
	return parseProto(&peeker{s: scanner.New(r)})
}

// proto = syntax { import | package | option |  message | enum | service | emptyStatement }
func parseProto(p *peeker) (proto *Proto, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", rec)
		}
	}()

	proto = &Proto{Syntax: parseSyntax(p)}
	for {
		switch next := p.peek(); next.Kind {
		case token.Package:
			if len(proto.Package.Identifier) > 0 {
				panicf("found second package definition")
			}
			proto.Package = parsePackage(p)
		case token.Import:
			proto.Imports = append(proto.Imports, parseImport(p))
		case token.Option:
			proto.Options = append(proto.Options, parseOption(p))
		case token.Message:
			proto.Messages = append(proto.Messages, parseMessage(p))
		case token.Enum:
			proto.Enums = append(proto.Enums, parseEnum(p))
		case token.Service:
			proto.Services = append(proto.Services, parseService(p))
		case token.Semicolon, token.Comment:
			// we ignore comments and empty statements
			continue
		case token.EOF:
			return proto, nil
		default:
			panicf("unexpected %s at top level definition", next)
		}
	}
}

type Syntax struct{ Value scanner.Token }

// syntax = "syntax" "=" quote "proto3" quote ";"
func parseSyntax(p *peeker) Syntax {
	p.consume(token.Syntax)
	p.consume(token.Equals)
	value := p.consume(token.StringLiteral)
	if text := value.Text; text != `"proto3"` && text != `'proto3'` {
		panicf("expected literal string \"proto3\", got %s instead", text)
	}
	p.consume(token.Semicolon)
	return Syntax{value}
}

type Import struct {
	Modifier scanner.Token
	Path     scanner.Token
}

// import = "import" [ "weak" | "public" ] strLit ";"
func parseImport(p *peeker) Import {
	p.consume(token.Import)
	var mod scanner.Token
	if tok, ok := p.maybeConsume(token.Weak, token.Public); ok {
		mod = tok
	}
	path := p.consume(token.StringLiteral)
	p.consume(token.Semicolon)
	return Import{Modifier: mod, Path: path}
}

type Package struct {
	Identifier FullIdentifier
}

// package = "package" fullIdent ";"
func parsePackage(p *peeker) Package {
	p.consume(token.Package)
	ident := parseFullIdentifier(p)
	p.consume(token.Semicolon)
	return Package{ident}
}

type Option struct {
	Prefix FullIdentifier // Parenthesised part of the identifier, if any.
	Name   FullIdentifier
	Value  interface{}
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

	if p.peek().Kind == token.Identifier {
		opt.Name = parseFullIdentifier(p)
	}

	if opt.Prefix == nil && opt.Name == nil {
		panicf("missing name in option")
	}

	p.consume(token.Equals)
	opt.Value = parseConstant(p)
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

type Message struct {
	Name      scanner.Token
	Fields    []Field
	Enums     []Enum
	Messages  []Message
	Options   []Option
	OneOfs    []OneOf
	Maps      []Map
	Reserveds []Reserved
}

// message = "message" messageName messageBody
// messageBody = "{" { field | enum | message | option | oneof | mapField | reserved | emptyStatement } "}"
func parseMessage(p *peeker) Message {
	p.consume(token.Message)
	msg := Message{Name: p.consume(token.Identifier)}
	p.consume(token.OpenBrace)

	for {
		switch kind := p.peek().Kind; {
		case token.IsType(kind) || kind == token.Identifier || kind == token.Repeated:
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

type Field struct {
	Repeated bool
	Type     scanner.Token
	Name     scanner.Token
	Number   scanner.Token
	Options  []Option
}

// field = [ "repeated" ] type fieldName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseField(p *peeker) Field {
	_, repeated := p.maybeConsume(token.Repeated)
	f := parseOneOfField(p)
	return Field{Repeated: repeated, Type: f.Type, Name: f.Name, Number: f.Number, Options: f.Options}
}

type Enum struct {
	Name    scanner.Token
	Fields  []EnumField
	Options []Option
}

// enum = "enum" enumName "{" { option | enumField | emptyStatement } "}"
func parseEnum(p *peeker) Enum {
	p.consume(token.Enum)
	enum := Enum{Name: p.consume(token.Identifier)}
	p.consume(token.OpenBrace)

	for {
		switch kind := p.peek().Kind; {
		case token.IsType(kind) || kind == token.Identifier || kind == token.Repeated:
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

type EnumField struct {
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

// enumField = ident "=" intLit fieldOptions ";"
func parseEnumField(p *peeker) EnumField {
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return EnumField{Name: name, Number: number, Options: opts}
}

type OneOf struct {
	Name   scanner.Token
	Fields []OneOfField
}

// oneof = "oneof" oneofName "{" { oneofField | emptyStatement } "}"
func parseOneOf(p *peeker) OneOf {
	p.consume(token.Oneof)
	o := OneOf{Name: p.consume(token.Identifier)}
	p.consume(token.OpenBrace)

	for {
		if _, ok := p.maybeConsume(token.CloseBrace); ok {
			return o
		}
		o.Fields = append(o.Fields, parseOneOfField(p))
	}
}

type OneOfField struct {
	Type    scanner.Token
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

// oneofField = type fieldName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseOneOfField(p *peeker) OneOfField {
	typ := p.scan()
	if typ.Kind != token.Identifier && !token.IsType(typ.Kind) {
		panicf("expected field type, got %s", typ)
	}
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return OneOfField{Type: typ, Name: name, Number: number, Options: opts}
}

type Map struct {
	KeyType   scanner.Token
	ValueType Type
	Name      scanner.Token
	Number    scanner.Token
	Options   []Option
}

// mapField = "map" "<" keyType "," type ">" mapName "=" fieldNumber [ "[" fieldOptions "]" ] ";"
func parseMap(p *peeker) Map {
	p.consume(token.Map)
	p.consume(token.OpenAngled)
	key := p.scan()
	if !token.IsKeyType(key.Kind) {
		panicf("expected key type, got %s", key)
	}
	p.consume(token.Comma)
	value := parseType(p)
	p.consume(token.CloseAngled)
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseFieldOptions(p)
	p.consume(token.Semicolon)

	return Map{KeyType: key, ValueType: value, Name: name, Number: number, Options: opts}
}

// Type contains either a predefined type in the form a Token,
// or a full identifier.
type Type struct {
	Predefined  scanner.Token
	UserDefined FullIdentifier
}

// type = "double" | "float" | "int32" | "int64" | "uint32" | "uint64"
//       | "sint32" | "sint64" | "fixed32" | "fixed64" | "sfixed32" | "sfixed64"
//       | "bool" | "string" | "bytes" | messageType | enumType
func parseType(p *peeker) Type {
	if token.IsType(p.peek().Kind) {
		return Type{Predefined: p.scan()}
	}
	return Type{UserDefined: parseFullIdentifier(p)}
}

type Reserved struct {
	IDs    []scanner.Token
	Names  []scanner.Token
	Ranges []Range
}

type Range struct{ From, To scanner.Token }

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
			res.Ranges = append(res.Ranges, Range{from, p.consume(from.Kind)})
		} else {
			if from.Kind == token.DecimalLiteral {
				res.IDs = append(res.IDs, from)
			} else {
				res.Names = append(res.Names, from)
			}
		}
		if _, ok := p.maybeConsume(token.Semicolon); ok {
			return res
		}
		p.consume(token.Comma)
	}
}

type Service struct {
	Name    scanner.Token
	Options []Option
	RPCs    []RPC
}

// service = "service" serviceName "{" { option | rpc | emptyStatement } "}"
func parseService(p *peeker) Service {
	p.consume(token.Service)
	svc := Service{Name: p.consume(token.Identifier)}

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

type RPC struct {
	Name    scanner.Token
	In      RPCParam
	Out     RPCParam
	Options []Option
}

// rpc = "rpc" rpcName rpcParam "returns" rpcParam (( "{" {option | emptyStatement } "}" ) | ";")
func parseRPC(p *peeker) RPC {
	p.consume(token.RPC)
	rpc := RPC{Name: p.consume(token.Identifier)}
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

type RPCParam struct {
	Stream bool
	Type   FullIdentifier
}

// rpcParam = "(" [ "stream" ] messageType ")"
func parseRPCParam(p *peeker) RPCParam {
	p.consume(token.OpenParen)
	_, stream := p.maybeConsume(token.Stream)
	typ := parseFullIdentifier(p)
	p.consume(token.CloseParen)
	return RPCParam{Stream: stream, Type: typ}
}

type FullIdentifier []scanner.Token

func parseFullIdentifier(p *peeker) FullIdentifier {
	ident := FullIdentifier{p.consume(token.Identifier)}
	for {
		dot := p.peek()
		if dot.Kind != token.Dot {
			return ident
		}
		p.scan()
		ident = append(ident, p.consume(token.Identifier))
	}
}

type SignedNumber struct {
	Sign, Number scanner.Token
}

func parseConstant(p *peeker) interface{} {
	switch next := p.peek(); {
	case token.IsConstant(next.Kind):
		return p.scan()
	case next.Kind == token.Plus || next.Kind == token.Minus:
		p.scan()
		number := p.scan()
		if !token.IsNumber(number.Kind) {
			panicf("expected number after %v, got %v", next.Kind, number.Kind)
		}
		return SignedNumber{next, number}
	case next.Kind == token.Identifier:
		return parseFullIdentifier(p)
	default:
		panicf("expected a valid constant value, but got %s", next)
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
	for tok.Kind == token.Comment {
		tok = p.s.Scan()
	}
	return tok
}

func (p *peeker) peek() (res scanner.Token) {
	if tok := p.peeked; tok != nil {
		return *tok
	}
	tok := p.s.Scan()
	for tok.Kind == token.Comment {
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
