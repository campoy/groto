package parser

import (
	"fmt"
	"io"
	"reflect"

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
	return parse(r)
}

func parse(r io.Reader) (proto *Proto, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%v", err)
		}
	}()

	proto = new(Proto)
	proto.parse(&peeker{s: scanner.New(r)})
	return proto, nil
}

type parser interface {
	parse(*peeker)
}

func panicf(format string, args ...interface{}) { panic(fmt.Sprintf(format, args...)) }

func (proto *Proto) parse(p *peeker) {
	for {
		switch next := p.peek(); next.Kind {
		case token.Illegal:
			panicf("unexpected %s", next.Text)
		case token.EOF:
			return
		case token.Semicolon, token.Comment:
			// we ignore comments and empty statements for now
			continue
		case token.Syntax:
			proto.Syntax.parse(p)
		case token.Import:
			list{&proto.Imports}.parse(p)
		case token.Package:
			list{&proto.Packages}.parse(p)
		case token.Option:
			list{&proto.Options}.parse(p)
		case token.Message:
			list{&proto.Messages}.parse(p)
		default:
			panicf("unexpected %v (%s) at top level definition", next.Kind, next.Text)
		}
	}
}

type Syntax struct{ Value scanner.Token }

func (s *Syntax) parse(p *peeker) {
	p.consume(token.Syntax)
	p.consume(token.Equals)
	s.Value = p.consume(token.StringLiteral)
	if text := s.Value.Text; text != `"proto3"` && text != `'proto3'` {
		panicf("expected literal string \"proto3\", got %s instead", text)
	}
	p.consume(token.Semicolon)
}

type Import struct {
	Modifier scanner.Token
	Path     scanner.Token
}

func (imp *Import) parse(p *peeker) {
	p.consume(token.Import)
	imp.Modifier = p.consumeIf(token.Weak, token.Public)
	imp.Path = p.consume(token.StringLiteral)
}

type Package struct {
	Identifier FullIdentifier
}

func (pkg *Package) parse(p *peeker) {
	p.consume(token.Package)
	pkg.Identifier.parse(p)
	p.consume(token.Semicolon)
}

type Option struct {
	Prefix *FullIdentifier
	Name   *FullIdentifier
	Value  Constant
}

func (opt *Option) parse(p *peeker) {
	p.consume(token.Option)
	(*shortOption)(opt).parse(p)
	p.consume(token.Semicolon)
}

type shortOption Option

func (opt *shortOption) parse(p *peeker) {
	if p.peek().Kind == token.OpenParens {
		p.scan()
		opt.Prefix = new(FullIdentifier)
		opt.Prefix.parse(p)
		p.consume(token.CloseParens)
	}

	if p.peek().Kind == token.Identifier {
		opt.Name = new(FullIdentifier)
		opt.Name.parse(p)
	}

	if opt.Prefix == nil && opt.Name == nil {
		panicf("missing name in option")
	}

	p.consume(token.Equals)
	opt.Value.parse(p)
}

type optionList struct{ s *[]Option }

func (l optionList) parse(p *peeker) {
	if p.peek().Kind != token.OpenBrackets {
		return
	}
	p.scan()

	for {
		var opt shortOption
		opt.parse(p)
		*l.s = append(*l.s, Option(opt))
		if p.peek().Kind == token.CloseBracket {
			p.scan()
			return
		}
		p.consume(token.Comma)
	}
}

type Message struct {
	Name scanner.Token
	Def  MessageDef
}

func (msg *Message) parse(p *peeker) {
	p.consume(token.Message)
	msg.Name = p.consume(token.Identifier)
	msg.Def.parse(p)
}

// MessageDef can be Field, Enum, Message, Option, Oneof, Mapfield, Reserved, or nil.
type MessageDef struct {
	Fields   []Field
	Enums    []Enum
	Messages []Message
	Options  []Option
	OneOfs   []OneOf
}

func (def *MessageDef) parse(p *peeker) {
	p.consume(token.OpenBraces)

	for {
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return
		case token.Enum:
			list{&def.Enums}.parse(p)
		case token.Message:
			list{&def.Messages}.parse(p)
		case token.Option:
			list{&def.Options}.parse(p)
		case token.Oneof:
			list{&def.OneOfs}.parse(p)
		case token.Map:
		// target = &def.MapFields
		case token.Reserved:
		// target = &def.Reserveds
		case token.Semicolon:
		case token.Identifier, token.Repeated:
			list{&def.Fields}.parse(p)
		default:
			if !token.IsType(next.Kind) {
				panicf("expected '}' to end message definition, got %s", next)
			}
			list{&def.Fields}.parse(p)
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

func (f *Field) parse(p *peeker) {
	if p.consumeIf(token.Repeated).Kind != token.Illegal {
		f.Repeated = true
	}

	f.Type = p.scan()
	if f.Type.Kind != token.Identifier && !token.IsType(f.Type.Kind) {
		panicf("expected field type, got %s", f.Type)
	}

	f.Name = p.consume(token.Identifier)
	p.consume(token.Equals)
	f.Number = p.consume(token.DecimalLiteral)
	(optionList{&f.Options}).parse(p)
	p.consume(token.Semicolon)
}

type Enum struct {
	Name scanner.Token
	Def  EnumDef
}

func (enum *Enum) parse(p *peeker) {
	p.consume(token.Enum)
	enum.Name = p.consume(token.Identifier)
	enum.Def.parse(p)
}

type EnumDef struct {
	Fields  []EnumField
	Options []Option
}

func (def *EnumDef) parse(p *peeker) {
	p.consume(token.OpenBraces)

	for {
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return
		case token.Option:
			list{&def.Options}.parse(p)
		case token.Identifier, token.Repeated:
			list{&def.Fields}.parse(p)
		default:
			if !token.IsType(next.Kind) {
				panicf("expected '}' to end message definition, got %s", next)
			}
			list{&def.Fields}.parse(p)
		}
	}
}

type EnumField struct {
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

func (f *EnumField) parse(p *peeker) {
	f.Name = p.consume(token.Identifier)
	p.consume(token.Equals)
	f.Number = p.consume(token.DecimalLiteral)
	(optionList{&f.Options}).parse(p)
	p.consume(token.Semicolon)
}

type OneOf struct {
	Name   scanner.Token
	Fields []Field
}

func (o *OneOf) parse(p *peeker) {
	p.consume(token.Oneof)
	o.Name = p.consume(token.Identifier)
	p.consume(token.OpenBraces)

	for {
		if p.peek().Kind == token.CloseBraces {
			return
		}
		(list{&o.Fields}).parse(p)
		if f := o.Fields[len(o.Fields)-1]; f.Repeated {
			panicf("required field %s not allowed inside of oneof", f.Name.Text)
		}
	}
}

type FullIdentifier struct {
	Identifiers []scanner.Token
}

func (ident *FullIdentifier) parse(p *peeker) {
	next := p.consume(token.Identifier)
	ident.Identifiers = append(ident.Identifiers, next)
	for {
		dot := p.peek()
		if dot.Kind != token.Dot {
			return
		}
		p.scan()
		ident.Identifiers = append(ident.Identifiers, p.consume(token.Identifier))
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
		var ident FullIdentifier
		ident.parse(p)
		c.Value = ident
	default:
		panicf("expected a valid constant value, but got %s", next)
	}
}

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

type list struct{ s interface{} }

func (a list) parse(p *peeker) {
	s := reflect.ValueOf(a.s).Elem()
	v := reflect.New(s.Type().Elem())
	v.Interface().(parser).parse(p)
	s.Set(reflect.Append(s, v.Elem()))
}
