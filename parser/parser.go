package parser

import (
	"fmt"
	"io"

	"github.com/campoy/groto/scanner"
)

type Proto struct {
	Syntax   *Syntax
	Imports  []*Import
	Packages []*Package
	Options  []*Option
}

var parseFuncs = map[string]func(*Proto, *parser) error{
	"import":  (*Proto).addImport,
	"package": (*Proto).addPackage,
	"option":  (*Proto).addOption,
}

func Parse(r io.Reader) (*Proto, error) {
	p := &parser{s: scanner.New(r)}

	syntax, err := p.parseSyntax()
	if err != nil {
		return nil, err
	}

	proto := &Proto{Syntax: syntax}

	for {
		switch next := p.Peek().(type) {
		case scanner.Error:
			return nil, next
		case scanner.EOF:
			return proto, nil
		case scanner.Punctuation:
			if next.Value == scanner.Semicolon {
				// empty statement
			} else {
				return nil, fmt.Errorf("unexpected %s", next)
			}
		case scanner.Comment:
			p.Scan()
		case scanner.Keyword:
			parseFunc, ok := parseFuncs[next.String()]
			if !ok {
				return nil, fmt.Errorf("unexpected keyword at top level definition %q", next)
			}
			if err := parseFunc(proto, p); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected %T %q", next, next)
		}
	}
}

type Syntax struct{ Value string }

func (p *parser) parseSyntax() (*Syntax, error) {
	p.consumeKeyword("syntax")
	p.consumePunctuation(scanner.Equal)
	p.consumeString("proto3")
	p.consumePunctuation(scanner.Semicolon)
	return &Syntax{Value: "proto3"}, p.err
}

type Import struct {
	Modifier string
	Path     string
}

func (proto *Proto) addImport(p *parser) error {
	imp, err := p.parseImport()
	if err != nil {
		return err
	}
	proto.Imports = append(proto.Imports, imp)
	return nil
}

func (p *parser) parseImport() (*Import, error) {
	p.consumeKeyword("import")

	modifier := ""
	next := p.Scan()
	switch next.(type) {
	case scanner.Keyword:
		s := next.String()
		if s != "weak" && s != "public" {
			return nil, fmt.Errorf("unexpected keyword %q", s)
		}
		modifier = s
		next = p.Scan()
	case scanner.String:
	default:
		return nil, fmt.Errorf("unexpected %q", next)
	}

	path, ok := next.(scanner.String)
	if !ok {
		return nil, fmt.Errorf("incorrect import path; got %q", next)
	}
	return &Import{Modifier: modifier, Path: path.String()}, nil
}

type Package struct {
	Identifier scanner.Token
}

func (proto *Proto) addPackage(p *parser) error {
	pkg, err := p.parsePackage()
	if err != nil {
		return err
	}
	proto.Packages = append(proto.Packages, pkg)
	return nil
}

func (p *parser) parsePackage() (*Package, error) {
	p.consumeKeyword("package")
	ident := p.Scan()
	switch ident.(type) {
	case scanner.Identifier, scanner.FullIdentifier:
	default:
		return nil, fmt.Errorf("expected package identifier, got %q", ident)
	}
	p.consumePunctuation(scanner.Semicolon)
	return &Package{Identifier: ident}, p.err
}

type Option struct {
	Name  scanner.Token
	Value scanner.Token
}

func (proto *Proto) addOption(p *parser) error {
	opt, err := p.parseOption()
	if err != nil {
		return err
	}
	proto.Options = append(proto.Options, opt)
	return nil
}

func (p *parser) parseOption() (*Option, error) {
	p.consumeKeyword("option")
	name := p.Scan()
	p.consumePunctuation(scanner.Equal)
	tok := p.Scan()
	switch tok.(type) {
	// supported kinds of options
	case scanner.FullIdentifier, scanner.Identifier,
		scanner.Boolean, scanner.String, scanner.Number, scanner.Integer:
	default:
		return nil, fmt.Errorf("unsupported option value %T %q", tok, tok)
	}
	return &Option{name, tok}, nil
}

// Parsing functions to make the code above much nicer.
// Not proud of this, but what's a gopher to do?

type parser struct {
	s      *scanner.Scanner
	peeked scanner.Token
	err    error
}

func (p *parser) Scan() scanner.Token {
	if tok := p.peeked; tok != nil {
		p.peeked = nil
		//logrus.Infof("scan from peek: %v", tok)
		return tok
	}
	tok := p.s.Scan()
	//logrus.Infof("scan from scanner: %v", tok)
	return tok
}

func (p *parser) Peek() scanner.Token {
	if tok := p.peeked; tok != nil {
		//logrus.Infof("peek from peek: %v", tok)
		return tok
	}
	p.peeked = p.s.Scan()
	//logrus.Infof("peek from scan: %v", p.peeked)
	return p.peeked
}

func (p *parser) consumeKeyword(value string) {
	if p.err != nil {
		return
	}
	tok := p.Scan()
	if _, ok := tok.(scanner.Keyword); !ok || tok.String() != value {
		p.err = fmt.Errorf("expected keyword %q, got %q", value, tok)
	}
}

func (p *parser) consumePunctuation(value rune) {
	if p.err != nil {
		return
	}
	next := p.Scan()
	tok, ok := next.(scanner.Punctuation)
	if !ok || tok.Value != value {
		p.err = fmt.Errorf("expected punctuation %q, got %q", value, next)
	}
}

func (p *parser) consumeString(value string) {
	text, err := p.readString()
	if err != nil {
		return
	}
	if text != value {
		p.err = fmt.Errorf("expected string %q, got %q", value, text)
	}
}

func (p *parser) readString() (string, error) {
	if p.err != nil {
		return "", p.err
	}
	tok := p.Scan()
	if _, ok := tok.(scanner.String); !ok {
		p.err = fmt.Errorf("expected string literal, got %q", tok)
		return "", p.err
	}
	return tok.String(), nil
}

func (p *parser) readPunctuation() (rune, error) {
	if p.err != nil {
		return 0, p.err
	}
	tok := p.Scan()
	if punc, ok := tok.(scanner.Punctuation); ok {
		return punc.Value, nil
	}
	p.err = fmt.Errorf("expected string literal, got %q", tok)
	return 0, p.err
}

func (p *parser) readIdentifier() (string, error) {
	if p.err != nil {
		return "", p.err
	}
	tok := p.Scan()
	if _, ok := tok.(scanner.Identifier); !ok {
		p.err = fmt.Errorf("expected identifier, got %q", tok)
		return "", p.err
	}
	return tok.String(), nil
}

func (p *parser) readFullIdentifier() ([]string, error) {
	ident, _ := p.readIdentifier()
	idents := []string{ident}
	for {
		if punc, ok := p.Peek().(scanner.Punctuation); !ok || punc.Value != scanner.Dot {
			return idents, nil
		}
		p.Scan()

		ident, err := p.readIdentifier()
		if err != nil {
			return nil, err
		}
		idents = append(idents, ident)
	}
}
