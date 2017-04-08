package parser

import (
	"fmt"
	"io"
	"strings"

	"bytes"

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

func parseProto(p *peeker) (proto *Proto, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", rec)
		}
	}()

	proto = new(Proto)
	for {
		switch next := p.peek(); next.Kind {
		case token.Illegal:
			panicf("unexpected %s", next.Text)
		case token.EOF:
			return proto, nil
		case token.Semicolon, token.Comment:
			// we ignore comments and empty statements for now
			continue
		case token.Syntax:
			proto.Syntax = parseSyntax(p)
		case token.Package:
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
		default:
			buf := new(bytes.Buffer)
			for p.peek().Kind != token.EOF {
				fmt.Fprintln(buf, p.scan())
			}
			panicf("unexpected %s at top level definition, right before %s", next, buf)
		}
	}
}

type Syntax struct{ Value scanner.Token }

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

func parsePackage(p *peeker) Package {
	p.consume(token.Package)
	ident := parseFullIdentifier(p)
	p.consume(token.Semicolon)
	return Package{ident}
}

type Option struct {
	Prefix *FullIdentifier
	Name   *FullIdentifier
	Value  Constant
}

func parseOption(p *peeker) Option {
	p.consume(token.Option)
	opt := parseShortOption(p)
	p.consume(token.Semicolon)
	return opt
}

type shortOption Option

func parseShortOption(p *peeker) Option {
	var opt Option
	if p.peek().Kind == token.OpenParens {
		p.scan()
		ident := parseFullIdentifier(p)
		opt.Prefix = &ident
		p.consume(token.CloseParens)
	}

	if p.peek().Kind == token.Identifier {
		ident := parseFullIdentifier(p)
		opt.Name = &ident
	}

	if opt.Prefix == nil && opt.Name == nil {
		panicf("missing name in option")
	}

	p.consume(token.Equals)
	opt.Value.parse(p)
	return opt
}

func parseOptionList(p *peeker) []Option {
	if p.peek().Kind != token.OpenBrackets {
		return nil
	}
	p.scan()

	var opts []Option
	for {
		opts = append(opts, parseShortOption(p))
		if p.peek().Kind == token.CloseBracket {
			p.scan()
			return opts
		}
		p.consume(token.Comma)
	}
}

type Message struct {
	Name scanner.Token
	Def  MessageDef
}

func parseMessage(p *peeker) Message {
	p.consume(token.Message)
	return Message{
		Name: p.consume(token.Identifier),
		Def:  parseMessageDef(p),
	}
}

type MessageDef struct {
	Fields    []Field
	Enums     []Enum
	Messages  []Message
	Options   []Option
	OneOfs    []OneOf
	Maps      []Map
	Reserveds []Reserved
}

func parseMessageDef(p *peeker) MessageDef {
	p.consume(token.OpenBraces)

	var def MessageDef

	for {
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return def
		case token.Enum:
			def.Enums = append(def.Enums, parseEnum(p))
		case token.Message:
			def.Messages = append(def.Messages, parseMessage(p))
		case token.Option:
			def.Options = append(def.Options, parseOption(p))
		case token.Oneof:
			def.OneOfs = append(def.OneOfs, parseOneOf(p))
		case token.Map:
			def.Maps = append(def.Maps, parseMap(p))
		case token.Reserved:
			def.Reserveds = append(def.Reserveds, parseReserved(p))
		case token.Semicolon:
			p.scan()
		case token.Identifier, token.Repeated:
			def.Fields = append(def.Fields, parseField(p))
		default:
			if !token.IsType(next.Kind) {
				panicf("expected '}' to end message definition, got %s", next)
			}
			def.Fields = append(def.Fields, parseField(p))
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

func parseField(p *peeker) Field {
	_, repeated := p.maybeConsume(token.Repeated)
	typ := p.scan()
	if typ.Kind != token.Identifier && !token.IsType(typ.Kind) {
		panicf("expected field type, got %s", typ)
	}
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseOptionList(p)
	p.consume(token.Semicolon)

	return Field{Repeated: repeated, Type: typ, Name: name, Number: number, Options: opts}
}

type Enum struct {
	Name scanner.Token
	Def  EnumDef
}

func parseEnum(p *peeker) Enum {
	p.consume(token.Enum)
	return Enum{
		Name: p.consume(token.Identifier),
		Def:  parseEnumDef(p),
	}
}

type EnumDef struct {
	Fields  []EnumField
	Options []Option
}

func parseEnumDef(p *peeker) EnumDef {
	p.consume(token.OpenBraces)
	var def EnumDef
	for {
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return def
		case token.Option:
			def.Options = append(def.Options, parseOption(p))
		case token.Identifier, token.Repeated:
			def.Fields = append(def.Fields, parseEnumField(p))
		default:
			if !token.IsType(next.Kind) {
				panicf("expected '}' to end message definition, got %s", next)
			}
			def.Fields = append(def.Fields, parseEnumField(p))
		}
	}
}

type EnumField struct {
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

func parseEnumField(p *peeker) EnumField {
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseOptionList(p)
	p.consume(token.Semicolon)

	return EnumField{Name: name, Number: number, Options: opts}
}

type OneOf struct {
	Name   scanner.Token
	Fields []Field
}

func parseOneOf(p *peeker) OneOf {
	p.consume(token.Oneof)
	o := OneOf{Name: p.consume(token.Identifier)}
	p.consume(token.OpenBraces)

	for {
		if _, ok := p.maybeConsume(token.CloseBraces); ok {
			return o
		}
		f := parseField(p)
		if f.Repeated {
			panicf("required field %s not allowed inside of oneof", f.Name.Text)
		}
		o.Fields = append(o.Fields, f)
	}
}

type Map struct {
	KeyType   scanner.Token
	ValueType FullIdentifier
	Name      scanner.Token
	Number    scanner.Token
	Options   []Option
}

func parseMap(p *peeker) Map {
	p.consume(token.Map)
	p.consume(token.OpenAngled)
	key := p.scan()
	if !token.IsKeyType(key.Kind) {
		panicf("expected key type, got %s", key)
	}
	p.consume(token.Comma)
	value := parseFullIdentifier(p)
	p.consume(token.CloseAngled)
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseOptionList(p)
	p.consume(token.Semicolon)

	return Map{KeyType: key, ValueType: value, Name: name, Number: number, Options: opts}
}

type Reserved struct {
	IDs    []scanner.Token
	Names  []scanner.Token
	Ranges []Range
}

type Range struct {
	From, To scanner.Token
}

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
		if tok, ok := p.maybeConsume(token.Comma, token.Semicolon); !ok {
			panicf("expected comma or semicolon in reserved list, got %s", tok)
		} else {
			if tok.Kind == token.Semicolon {
				return res
			}
		}
	}
}

type Service struct {
	Name    scanner.Token
	Options []Option
	RPCs    []RPC
}

func parseService(p *peeker) Service {
	p.consume(token.Service)
	svc := Service{Name: p.consume(token.Identifier)}

	p.consume(token.OpenBraces)
	for {
		switch p.peek().Kind {
		case token.CloseBraces:
			p.scan()
			return svc
		case token.Option:
			svc.Options = append(svc.Options, parseOption(p))
		case token.RPC:
			svc.RPCs = append(svc.RPCs, parseRPC(p))
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

func parseRPC(p *peeker) RPC {
	p.consume(token.RPC)
	rpc := RPC{Name: p.consume(token.Identifier)}
	rpc.In = parseRPCParam(p)
	p.consume(token.Returns)
	rpc.Out = parseRPCParam(p)

	if _, ok := p.maybeConsume(token.Semicolon); ok {
		return rpc
	}

	p.consume(token.OpenBraces)
	for {
		if _, ok := p.maybeConsume(token.CloseBraces); ok {
			return rpc
		}
		rpc.Options = append(rpc.Options, parseOption(p))
	}
}

type RPCParam struct {
	Stream bool
	Type   FullIdentifier
}

func parseRPCParam(p *peeker) RPCParam {
	p.consume(token.OpenParens)
	_, stream := p.maybeConsume(token.Stream)
	typ := parseFullIdentifier(p)
	p.consume(token.CloseParens)
	return RPCParam{Stream: stream, Type: typ}
}

type FullIdentifier struct {
	Identifiers []scanner.Token
}

func parseFullIdentifier(p *peeker) FullIdentifier {
	idents := []scanner.Token{p.consume(token.Identifier)}
	for {
		dot := p.peek()
		if dot.Kind != token.Dot {
			return FullIdentifier{idents}
		}
		p.scan()
		idents = append(idents, p.consume(token.Identifier))
	}

}

type Constant struct {
	Value interface{}
}

type SignedNumber struct {
	Sign, Number scanner.Token
}

func (c *Constant) parse(p *peeker) {
	switch next := p.peek(); {
	case token.IsConstant(next.Kind):
		c.Value = p.scan()
	case next.Kind == token.Plus || next.Kind == token.Minus:
		p.scan()
		number := p.scan()
		if !token.IsNumber(number.Kind) {
			panicf("expected number after %v, got %v", next.Kind, number.Kind)
		}
		c.Value = SignedNumber{next, number}
	case next.Kind == token.Identifier:
		c.Value = parseFullIdentifier(p)
	default:
		panicf("expected a valid constant value, but got %s", next)
	}
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
