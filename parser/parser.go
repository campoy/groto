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
	p := &peeker{s: scanner.New(r)}

	var proto Proto
	if err := proto.parse(p); err != nil {
		return nil, err
	}
	return &proto, nil
}

type parser interface {
	parse(*peeker) error
}

func (proto *Proto) parse(p *peeker) error {
	for {
		var target parser
		switch next := p.peek(); next.Kind {
		case token.Illegal:
			return fmt.Errorf("unexpected %s", next.Text)
		case token.EOF:
			return nil
		case token.Semicolon, token.Comment:
			// we ignore comments and empty statements for now
			continue
		case token.Syntax:
			target = &proto.Syntax
		case token.Import:
			target = list{&proto.Imports}
		case token.Package:
			target = list{&proto.Packages}
		case token.Option:
			target = list{&proto.Options}
		case token.Message:
			target = list{&proto.Messages}
		default:
			return fmt.Errorf("unexpected %v (%s) at top level definition", next.Kind, next.Text)
		}
		if err := target.parse(p); err != nil {
			return err
		}
	}
}

type Syntax struct{ Value scanner.Token }

func (s *Syntax) parse(p *peeker) error {
	if tok, ok := p.consume(token.Syntax, token.Equals); !ok {
		return fmt.Errorf("expected '=', got %s instead", tok.Text)
	}
	tok := p.scan()
	if tok.Kind != token.StringLiteral {
		return fmt.Errorf("expected literal string \"proto3\", got a %v instead", tok.Kind)
	}
	if tok.Text != `"proto3"` && tok.Text != `'proto3'` {
		return fmt.Errorf("expected literal string \"proto3\", got %s instead", tok.Text)
	}
	s.Value = tok
	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of the syntax statement")
	}
	return nil
}

type Import struct {
	Modifier scanner.Token
	Path     scanner.Token
}

func (imp *Import) parse(p *peeker) error {
	if tok, ok := p.consume(token.Import); !ok {
		return fmt.Errorf("expected 'import' keyword, but instead got %s", tok)
	}
	next := p.scan()
	if next.Kind == token.Weak || next.Kind == token.Public {
		imp.Modifier = next
		next = p.scan()
	}
	if next.Kind != token.StringLiteral {
		return fmt.Errorf("expected imported package name, got %s", next)
	}
	imp.Path = next
	return nil
}

type Package struct {
	Identifier FullIdentifier
}

func (pkg *Package) parse(p *peeker) error {
	if tok, ok := p.consume(token.Package); !ok {
		return fmt.Errorf("expected keyword package, got %s", tok)
	}
	if err := pkg.Identifier.parse(p); err != nil {
		return err
	}
	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of package statement")
	}
	return nil
}

type Option struct {
	Prefix *FullIdentifier
	Name   *FullIdentifier
	Value  Constant
}

func (opt *Option) parse(p *peeker) error {
	if tok, ok := p.consume(token.Option); !ok {
		return fmt.Errorf("expected keyword option, got %s", tok)
	}
	if err := (*shortOption)(opt).parse(p); err != nil {
		return err
	}
	if tok, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of option statement, got %s", tok)
	}
	return nil
}

type shortOption Option

func (opt *shortOption) parse(p *peeker) error {
	next := p.peek()
	if next.Kind == token.OpenParens {
		p.scan()
		opt.Prefix = new(FullIdentifier)
		if err := opt.Prefix.parse(p); err != nil {
			return err
		}
		if tok, ok := p.consume(token.CloseParens); !ok {
			return fmt.Errorf("expected closing parenthesis after %s, got %s", opt.Prefix, tok)
		}
		next = p.scan()
	}

	if next.Kind == token.Identifier {
		opt.Name = new(FullIdentifier)
		if err := opt.Name.parse(p); err != nil {
			return err
		}
		next = p.scan()
	}

	if opt.Prefix == nil && opt.Name == nil {
		return fmt.Errorf("missing name in option")
	}

	if next.Kind != token.Equals {
		return fmt.Errorf("expected '=' between option name and value, got %s", next)
	}
	return opt.Value.parse(p)
}

type optionList struct{ s *[]Option }

func (l optionList) parse(p *peeker) error {
	if p.peek().Kind != token.OpenBrackets {
		return nil
	}

	p.scan()
	for {
		var opt shortOption
		if err := opt.parse(p); err != nil {
			return err
		}
		*l.s = append(*l.s, Option(opt))
		if p.peek().Kind == token.CloseBracket {
			p.scan()
			return nil
		}
		if tok, ok := p.consume(token.Comma); !ok {
			return fmt.Errorf("expected ',' in between options, got %s", tok)
		}
	}
}

type Message struct {
	Name scanner.Token
	Def  MessageDef
}

func (msg *Message) parse(p *peeker) error {
	if tok, ok := p.consume(token.Message); !ok {
		return fmt.Errorf("expected keyword message, got %s", tok)
	}
	name := p.scan()
	if name.Kind != token.Identifier {
		return fmt.Errorf("expected message identifier, got %s", name)
	}
	msg.Name = name
	return msg.Def.parse(p)
}

// MessageDef can be Field, Enum, Message, Option, Oneof, Mapfield, Reserved, or nil.
type MessageDef struct {
	Fields   []Field
	Enums    []Enum
	Messages []Message
	Options  []Option
	OneOfs   []OneOf
}

func (def *MessageDef) parse(p *peeker) error {
	if tok, ok := p.consume(token.OpenBraces); !ok {
		return fmt.Errorf("expected '{' to start message definition, got %s", tok)
	}

	for {
		var target parser
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return nil
		case token.Enum:
			target = list{&def.Enums}
		case token.Message:
			target = list{&def.Messages}
		case token.Option:
			target = list{&def.Options}
		case token.Oneof:
			target = list{&def.OneOfs}
		case token.Map:
		// target = &def.MapFields
		case token.Reserved:
		// target = &def.Reserveds
		case token.Semicolon:
		case token.Identifier, token.Repeated:
			target = list{&def.Fields}
		default:
			if !token.IsType(next.Kind) {
				return fmt.Errorf("expected '}' to end message definition, got %s", next)
			}
			target = list{&def.Fields}
		}
		if target != nil {
			if err := target.parse(p); err != nil {
				return err
			}
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

func (f *Field) parse(p *peeker) error {
	next := p.scan()
	if next.Kind == token.Repeated {
		f.Repeated = true
		next = p.scan()
	}
	if next.Kind != token.Identifier && !token.IsType(next.Kind) {
		return fmt.Errorf("expected field type, got %s", next)
	}
	f.Type = next
	next = p.scan()
	if next.Kind != token.Identifier {
		return fmt.Errorf("expected field name, got %s", next)
	}
	f.Name = next

	if tok, ok := p.consume(token.Equals); !ok {
		return fmt.Errorf("expected '=' after field name, got %s", tok)
	}

	number := p.scan()
	if number.Kind != token.DecimalLiteral {
		return fmt.Errorf("expected field number, got %s", number)
	}
	f.Number = number

	if err := (optionList{&f.Options}).parse(p); err != nil {
		return err
	}

	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of field definition")
	}
	return nil
}

type Enum struct {
	Name scanner.Token
	Def  EnumDef
}

func (enum *Enum) parse(p *peeker) error {
	if tok, ok := p.consume(token.Enum); !ok {
		return fmt.Errorf("expected keyword enum, got %s", tok)
	}

	enum.Name = p.scan()
	if enum.Name.Kind != token.Identifier {
		return fmt.Errorf("expected enum name, got %s", enum.Name)
	}

	return enum.Def.parse(p)
}

type EnumDef struct {
	Fields  []EnumField
	Options []Option
}

func (def *EnumDef) parse(p *peeker) error {
	if tok, ok := p.consume(token.OpenBraces); !ok {
		return fmt.Errorf("expected '{' to start message definition, got %s", tok)
	}

	for {
		var target parser
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return nil
		case token.Option:
			target = list{&def.Options}
		case token.Identifier, token.Repeated:
			target = list{&def.Fields}
		default:
			if !token.IsType(next.Kind) {
				return fmt.Errorf("expected '}' to end message definition, got %s", next)
			}
			target = list{&def.Fields}
		}
		if target != nil {
			if err := target.parse(p); err != nil {
				return err
			}
		}
	}
}

type EnumField struct {
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

func (f *EnumField) parse(p *peeker) error {
	next := p.scan()
	if next.Kind != token.Identifier {
		return fmt.Errorf("expected field name, got %s", next)
	}
	f.Name = next

	if tok, ok := p.consume(token.Equals); !ok {
		return fmt.Errorf("expected '=' after field name, got %s", tok)
	}

	number := p.scan()
	if number.Kind != token.DecimalLiteral {
		return fmt.Errorf("expected field number, got %s", number)
	}
	f.Number = number

	if err := (optionList{&f.Options}).parse(p); err != nil {
		return err
	}

	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of field definition")
	}
	return nil
}

type OneOf struct {
	Name   scanner.Token
	Fields []Field
}

func (o *OneOf) parse(p *peeker) error {
	if tok, ok := p.consume(token.Oneof); !ok {
		return fmt.Errorf("expected keyword enum, got %s", tok)
	}

	o.Name = p.scan()
	if o.Name.Kind != token.Identifier {
		return fmt.Errorf("expected oneof name, got %s", o.Name)
	}

	if tok, ok := p.consume(token.OpenBraces); !ok {
		return fmt.Errorf("expected '{' to begin oneoflist, got %s", tok)
	}

	for {
		if p.peek().Kind == token.CloseBraces {
			return nil
		}
		if err := (list{&o.Fields}).parse(p); err != nil {
			return err
		}
		if f := o.Fields[len(o.Fields)-1]; f.Repeated {
			return fmt.Errorf("required field %s not allowed inside of oneof", f.Name.Text)
		}
	}
}

type FullIdentifier struct {
	Identifiers []scanner.Token
}

func (ident *FullIdentifier) parse(p *peeker) error {
	next := p.scan()
	if next.Kind != token.Identifier {
		return fmt.Errorf("expected identifier, got %s", next)
	}

	ident.Identifiers = append(ident.Identifiers, next)
	for {
		dot := p.peek()
		if dot.Kind != token.Dot {
			return nil
		}
		p.scan()
		name := p.scan()
		if name.Kind != token.Identifier {
			return fmt.Errorf("expected identifier, got %s", next)
		}
		ident.Identifiers = append(ident.Identifiers, name)
	}
}

type Constant struct {
	Value interface{}
}

type SignedNumber struct {
	Sign, Number scanner.Token
}

func (c *Constant) parse(p *peeker) error {
	switch next := p.peek(); {
	case token.IsConstant(next.Kind):
		c.Value = p.scan()
	case next.Kind == token.Plus || next.Kind == token.Minus:
		p.scan()
		number := p.scan()
		if !token.IsNumber(number.Kind) {
			return fmt.Errorf("expected number after %v, got %v", next.Kind, number.Kind)
		}
		c.Value = SignedNumber{next, number}
	case next.Kind == token.Identifier:
		var ident FullIdentifier
		if err := ident.parse(p); err != nil {
			return err
		}
		c.Value = ident
	default:
		return fmt.Errorf("expected a valid constant value, but got %s", next)
	}
	return nil
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

func (p *peeker) consume(tokens ...token.Kind) (*scanner.Token, bool) {
	for _, tok := range tokens {
		got := p.scan()
		if got.Kind != tok {
			return &got, false
		}
	}
	return nil, true
}

type list struct{ s interface{} }

func (a list) parse(p *peeker) error {
	s := reflect.ValueOf(a.s).Elem()
	v := reflect.New(s.Type().Elem())
	if err := v.Interface().(parser).parse(p); err != nil {
		return err
	}
	s.Set(reflect.Append(s, v.Elem()))
	return nil
}
