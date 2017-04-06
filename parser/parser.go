package parser

import (
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

type Proto struct {
	Syntax   Syntax
	Imports  Imports
	Packages Packages
	Options  Options
	Messages Messages
}

func Parse(r io.Reader) (*Proto, error) {
	p := &parser{s: scanner.New(r)}

	var proto Proto
	if err := proto.parse(p); err != nil {
		return nil, err
	}
	return &proto, nil
}

type parsable interface {
	parse(*parser) error
}

func (proto *Proto) parse(p *parser) error {
	for {
		var target parsable
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
			target = &proto.Imports
		case token.Package:
			target = &proto.Packages
		case token.Option:
			target = &proto.Options
		case token.Message:
			target = &proto.Messages
		default:
			return fmt.Errorf("unexpected %v (%s) at top level definition", next.Kind, next.Text)
		}
		if err := target.parse(p); err != nil {
			return err
		}
	}
}

type Syntax struct{ Value scanner.Token }

func (s *Syntax) parse(p *parser) error {
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

type Imports []Import

func (imps *Imports) parse(p *parser) error {
	var imp Import
	if err := imp.parse(p); err != nil {
		return err
	}
	*imps = append(*imps, imp)
	return nil
}

type Import struct {
	Modifier scanner.Token
	Path     scanner.Token
}

func (imp *Import) parse(p *parser) error {
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

type Packages []Package

func (pkgs *Packages) parse(p *parser) error {
	var pkg Package
	if err := pkg.parse(p); err != nil {
		return err
	}
	*pkgs = append(*pkgs, pkg)
	return nil
}

type Package struct {
	Identifier FullIdentifier
}

func (pkg *Package) parse(p *parser) error {
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

type Options []Option

func (opts *Options) parse(p *parser) error {
	var opt Option
	if err := opt.parse(p); err != nil {
		return err
	}
	*opts = append(*opts, opt)
	return nil
}

type Option struct {
	Prefix *FullIdentifier
	Name   FullIdentifier
	Value  Constant
}

func (opt *Option) parse(p *parser) error {
	if tok, ok := p.consume(token.Option); !ok {
		return fmt.Errorf("expected keyword option, got %s", tok)
	}

	next := p.peek()
	if next.Kind == token.OpenParens {
		p.scan()
		if err := opt.Prefix.parse(p); err != nil {
			return err
		}
		if tok, ok := p.consume(token.CloseParens); !ok {
			return fmt.Errorf("expected closing parenthesis after %s, got %s", opt.Prefix, tok)
		}
		next = p.scan()
	}
	if next.Kind != token.Identifier {
		return fmt.Errorf("expected identifer in option name, got %v %s", next.Kind, next.Text)
	}
	if err := opt.Name.parse(p); err != nil {
		return err
	}

	if tok, ok := p.consume(token.Equals); !ok {
		return fmt.Errorf("expected '=' between option name and value, got %s", tok)
	}
	if err := opt.Value.parse(p); err != nil {
		return err
	}
	if tok, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of option statement, got %s", tok)
	}
	return nil
}

type Messages []Message

func (msgs *Messages) parse(p *parser) error {
	var msg Message
	if err := msg.parse(p); err != nil {
		return err
	}
	*msgs = append(*msgs, msg)
	return nil
}

type Message struct {
	Name scanner.Token
	Def  MessageDef
}

func (msg *Message) parse(p *parser) error {
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

type MessageDef struct {
	// can be Field, Enum, Message, Option, Oneof, Mapfield, Reserved, or nil
	Definitions []interface{}
}

func (def *MessageDef) parse(p *parser) error {
	if tok, ok := p.consume(token.OpenBraces); !ok {
		return fmt.Errorf("expected '{' to start message definition, got %s", tok)
	}

	for {
		var target parsable
		switch next := p.peek(); next.Kind {
		case token.CloseBraces:
			p.scan()
			return nil
		case token.Enum:
			// target = new(Enum)
		case token.Message:
			target = new(Message)
		case token.Option:
			target = new(Option)
		case token.Oneof:
		// target = new(Oneof)
		case token.Map:
		// target = new(MapField)
		case token.Reserved:
		// target = new(Reserved)
		case token.Semicolon:
		case token.Identifier, token.Repeated:
			target = new(Field)
		default:
			if !token.IsType(next.Kind) {
				return fmt.Errorf("expected '}' to end message definition, got %s", next)
			}
			target = new(Field)
		}
		if target != nil {
			if err := target.parse(p); err != nil {
				return err
			}
			def.Definitions = append(def.Definitions, target)
		}
	}
}

type Field struct {
	Repeated bool
	Type     scanner.Token
	Name     scanner.Token
	Number   scanner.Token
	Options  FieldOptions
}

func (f *Field) parse(p *parser) error {
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

	next = p.scan()
	if next.Kind == token.OpenBrackets {
		if err := f.Options.parse(p); err != nil {
			return nil
		}
		if tok, ok := p.consume(token.CloseBracket); !ok {
			return fmt.Errorf("expected '}' to close field options, got %s", tok)
		}
		next = p.scan()
	}
	if next.Kind != token.Semicolon {
		return fmt.Errorf("missing semicolon at the end of field definition")
	}
	return nil
}

type FieldOptions []FieldOption

func (ops *FieldOptions) parse(p *parser) error {
	return nil
}

type FieldOption struct {
	// TODO
}

// Parsing functions to make the code above much nicer.
// Not proud of this, but what's a gopher to do?

type FullIdentifier struct {
	Identifiers []scanner.Token
}

func (ident *FullIdentifier) parse(p *parser) error {
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

func (c *Constant) parse(p *parser) error {
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

type parser struct {
	s      *scanner.Scanner
	peeked *scanner.Token
}

func (p *parser) scan() (res scanner.Token) {
	defer func() { logrus.Infof("scan: %s", res) }()
	if tok := p.peeked; tok != nil {
		p.peeked = nil
		return *tok
	}
	tok := p.s.Scan()
	return tok
}

func (p *parser) peek() (res scanner.Token) {
	defer func() { logrus.Infof("peek: %s", res) }()
	if tok := p.peeked; tok != nil {
		return *tok
	}
	tok := p.s.Scan()
	p.peeked = &tok
	return tok
}

func (p *parser) consume(tokens ...token.Kind) (*scanner.Token, bool) {
	for _, tok := range tokens {
		got := p.scan()
		if got.Kind != tok {
			return &got, false
		}
	}
	return nil, true
}
