package parser

import (
	"fmt"
	"io"

	"github.com/campoy/groto/scanner"
	"github.com/campoy/groto/token"
)

var logf = func(format string, args ...interface{}) {}

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
	logf("> proto.parse")
	defer logf("< proto.parse")

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
			target = (*imports)(&proto.Imports)
		case token.Package:
			target = (*packages)(&proto.Packages)
		case token.Option:
			target = (*options)(&proto.Options)
		case token.Message:
			target = (*messages)(&proto.Messages)
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
	logf("> syntax.parse")
	defer logf("< syntax.parse")

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

type imports []Import

func (imps *imports) parse(p *peeker) error {
	logf("> imports.parse")
	defer logf("< imports.parse")

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

func (imp *Import) parse(p *peeker) error {
	logf("> import.parse")
	defer logf("< import.parse")

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

type packages []Package

func (pkgs *packages) parse(p *peeker) error {
	logf("> packages.parse")
	defer logf("< packages.parse")

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

func (pkg *Package) parse(p *peeker) error {
	logf("> package.parse")
	defer logf("< package.parse")

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

type options []Option

func (opts *options) parse(p *peeker) error {
	logf("> options.parse")
	defer logf("< options.parse")

	var opt Option
	if tok, ok := p.consume(token.Option); !ok {
		return fmt.Errorf("expected keyword option, got %s", tok)
	}

	if err := opt.parse(p); err != nil {
		return err
	}

	if tok, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of option statement, got %s", tok)
	}

	*opts = append(*opts, opt)
	return nil
}

type Option struct {
	Prefix *FullIdentifier
	Name   *FullIdentifier
	Value  Constant
}

func (opt *Option) parse(p *peeker) error {
	logf("> option.parse")
	defer logf("< option.parse")

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
	if err := opt.Value.parse(p); err != nil {
		return err
	}
	return nil
}

type messages []Message

func (msgs *messages) parse(p *peeker) error {
	logf("> messages.parse")
	defer logf("< messages.parse")

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

func (msg *Message) parse(p *peeker) error {
	logf("> message.parse")
	defer logf("< message.parse")

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
	logf("> messagedef.parse")
	defer logf("< messagedef.parse")

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
			target = (*enums)(&def.Enums)
		case token.Message:
			target = (*messages)(&def.Messages)
		case token.Option:
			target = (*options)(&def.Options)
		case token.Oneof:
			target = (*oneOfs)(&def.OneOfs)
		case token.Map:
		// target = &def.MapFields
		case token.Reserved:
		// target = &def.Reserveds
		case token.Semicolon:
		case token.Identifier, token.Repeated:
			target = (*fields)(&def.Fields)
		default:
			if !token.IsType(next.Kind) {
				return fmt.Errorf("expected '}' to end message definition, got %s", next)
			}
			target = (*fields)(&def.Fields)
		}
		if target != nil {
			if err := target.parse(p); err != nil {
				return err
			}
		}
	}
}

type fields []Field

func (fs *fields) parse(p *peeker) error {
	logf("> fields.parse")
	defer logf("< fields.parse")

	var f Field
	if err := f.parse(p); err != nil {
		return err
	}
	*fs = append(*fs, f)
	return nil
}

type Field struct {
	Repeated bool
	Type     scanner.Token
	Name     scanner.Token
	Number   scanner.Token
	Options  []Option
}

func (f *Field) parse(p *peeker) error {
	logf("> field.parse")
	defer logf("< field.parse")

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

	if err := (*fieldOptions)(&f.Options).parse(p); err != nil {
		return err
	}

	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of field definition")
	}
	return nil
}

type fieldOptions []Option

func (opts *fieldOptions) parse(p *peeker) error {
	if p.peek().Kind != token.OpenBrackets {
		return nil
	}
	p.scan()

	for {
		var opt Option
		if err := opt.parse(p); err != nil {
			return err
		}
		*opts = append(*opts, opt)
		if p.peek().Kind != token.Comma {
			break
		}
		p.scan()
	}

	if tok, ok := p.consume(token.CloseBracket); !ok {
		return fmt.Errorf("expected ']' to close field options, got %s", tok)
	}
	return nil
}

type enums []Enum

func (enums *enums) parse(p *peeker) error {
	var enum Enum
	if err := enum.parse(p); err != nil {
		return err
	}
	*enums = append(*enums, enum)
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
	logf("> enumdef.parse")
	defer logf("< enumdef.parse")

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
			target = (*options)(&def.Options)
		case token.Identifier, token.Repeated:
			target = (*enumFields)(&def.Fields)
		default:
			if !token.IsType(next.Kind) {
				return fmt.Errorf("expected '}' to end message definition, got %s", next)
			}
			target = (*enumFields)(&def.Fields)
		}
		if target != nil {
			if err := target.parse(p); err != nil {
				return err
			}
		}
	}
}

type enumFields []EnumField

func (fs *enumFields) parse(p *peeker) error {
	logf("> enumfields.parse")
	defer logf("< enumfields.parse")

	var f EnumField
	if err := f.parse(p); err != nil {
		return err
	}
	*fs = append(*fs, f)
	return nil
}

type EnumField struct {
	Name    scanner.Token
	Number  scanner.Token
	Options []Option
}

func (f *EnumField) parse(p *peeker) error {
	logf("> enumfield.parse")
	defer logf("< enumfield.parse")

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

	if err := (*fieldOptions)(&f.Options).parse(p); err != nil {
		return err
	}

	if _, ok := p.consume(token.Semicolon); !ok {
		return fmt.Errorf("missing semicolon at the end of field definition")
	}
	return nil
}

type oneOfs []OneOf

func (os *oneOfs) parse(p *peeker) error {
	logf("> oneofs.parse")
	defer logf("< oneofs.parse")

	var o OneOf
	if err := o.parse(p); err != nil {
		return err
	}
	*os = append(*os, o)
	return nil
}

type OneOf struct {
	Name   scanner.Token
	Fields []Field
}

func (o *OneOf) parse(p *peeker) error {
	logf("> oneof.parse")
	defer logf("< oneof.parse")

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
		if err := (*fields)(&o.Fields).parse(p); err != nil {
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
	logf("> fullidentifier.parse")
	defer logf("< fullidentifier.parse")

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
	logf("> constant.parse")
	defer logf("< constant.parse")

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
	defer func() { logf("scan: %s", res) }()
	if tok := p.peeked; tok != nil {
		p.peeked = nil
		return *tok
	}
	tok := p.s.Scan()
	return tok
}

func (p *peeker) peek() (res scanner.Token) {
	defer func() { logf("peek: %s", res) }()
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
