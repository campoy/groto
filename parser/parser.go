package parser

import (
	"fmt"
	"io"

	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

type Proto struct {
	Syntax   Syntax
	Imports  []Import
	Packages []Package
	Options  []Option
	Messages []Message
}

func Parse(r io.Reader) (*Proto, error) {
	return parseProto(&peeker{s: scanner.New(r)})
}

func parseProto(p *peeker) (proto *Proto, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", err)
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
		case token.Import:
			proto.Imports = append(proto.Imports, parseImport(p))
		case token.Package:
			proto.Packages = append(proto.Packages, parsePackage(p))
		case token.Option:
			proto.Options = append(proto.Options, parseOption(p))
		case token.Message:
			proto.Messages = append(proto.Messages, parseMessage(p))
		default:
			panicf("unexpected %v (%s) at top level definition", next.Kind, next.Text)
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
	return Import{
		Modifier: p.consumeIf(token.Weak, token.Public),
		Path:     p.consume(token.StringLiteral),
	}
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

// MessageDef can be Field, Enum, Message, Option, Oneof, Mapfield, Reserved, or nil.
type MessageDef struct {
	Fields   []Field
	Enums    []Enum
	Messages []Message
	Options  []Option
	OneOfs   []OneOf
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
		// target = &def.MapFields
		case token.Reserved:
		// target = &def.Reserveds
		case token.Semicolon:
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
	repeated := p.consumeIf(token.Repeated).Kind != token.Illegal
	typ := p.scan()
	if typ.Kind != token.Identifier && !token.IsType(typ.Kind) {
		panicf("expected field type, got %s", typ)
	}
	name := p.consume(token.Identifier)
	p.consume(token.Equals)
	number := p.consume(token.DecimalLiteral)
	opts := parseOptionList(p)
	p.consume(token.Semicolon)
	return Field{
		Repeated: repeated,
		Type:     typ,
		Name:     name,
		Number:   number,
		Options:  opts,
	}
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
	return EnumField{
		Name:    name,
		Number:  number,
		Options: opts,
	}
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
		if p.peek().Kind == token.CloseBraces {
			return o
		}
		f := parseField(p)
		if f.Repeated {
			panicf("required field %s not allowed inside of oneof", f.Name.Text)
		}
		o.Fields = append(o.Fields, f)
	}
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
	return tok
}

func (p *peeker) peek() (res scanner.Token) {
	if tok := p.peeked; tok != nil {
		return *tok
	}
	tok := p.s.Scan()
	p.peeked = &tok
	return tok
}

func (p *peeker) consume(tok token.Kind) scanner.Token {
	got := p.scan()
	if got.Kind != tok {
		panicf("expected %s, got %s", tok, got)
	}
	return got
}

func (p *peeker) consumeIf(toks ...token.Kind) scanner.Token {
	got := p.peek()
	for _, tok := range toks {
		if got.Kind == tok {
			return p.scan()
		}
	}
	return scanner.Token{}
}
