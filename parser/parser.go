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

// import = "import" [ "weak" | "public" ] strLit ";"
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

// package = "package" fullIdent ";"

type Package struct {
	Identifier []string
}

func (p *parser) parsePackage() (*Package, error) {
	p.consumeKeyword("package")
	ident, _ := p.readIdentifier()
	idents := []string{ident}
	for p.err == nil {
		punc, err := p.readPunctuation()
		if err != nil {
			return nil, err
		}
		if punc == scanner.Semicolon {
			return &Package{Identifier: idents}, nil
		}
		if punc != scanner.Dot {
			return nil, fmt.Errorf("unexpected punctuation %q", punc)
		}

		ident, _ := p.readIdentifier()
		idents = append(idents, ident)
	}
	return nil, p.err
}

func Parse(r io.Reader) (*Proto, error) {
	p := &parser{Scanner: scanner.New(r)}
	syntax, err := p.parseSyntax()
	if err != nil {
		return nil, err
	}

	proto := &Proto{
		Syntax: syntax,
	}
	for {
		switch next := p.Scan().(type) {
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
		case scanner.Keyword:
			switch next.String() {
			case "import":
				imp, err := p.parseImport()
				if err != nil {
					return nil, err
				}
				proto.Imports = append(proto.Imports, imp)
			case "package":
				pkg, err := p.parsePackage()
				if err != nil {
					return nil, err
				}
				proto.Packages = append(proto.Packages, pkg)
			case "option":
			case "message":
			case "enum":
			case "service":
			default:
				return nil, fmt.Errorf("unexpected keyword %s", next)
			}
		default:
			return nil, fmt.Errorf("unexpected %s", next)
		}
	}
}

type parser struct {
	*scanner.Scanner
	err error
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
