package main

import (
	"fmt"
	"strconv"
)

type parser struct {
	lex       *lexer
	tok       token
	def       *cdlTop
	lastCmd   *command
	lastTopic *topic
	pathBuf   [8]string
}

func parseDef(dsl string) (*cdlTop, error) {
	p := &parser{lex: newLexer(dsl), def: &cdlTop{GlobalParams: map[string]value{}}}
	p.next()
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.def, nil
}

func (p *parser) next() {
	p.tok = p.lex.nextToken()
}

func (p *parser) parse() error {
	for p.tok.kind != tokEOF {
		if p.tok.kind == tokError {
			return fmt.Errorf("line %d: %s", p.tok.line, p.tok.value)
		}
		if err := p.parseStatement(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseStatement() error {
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected keyword, got %v", p.tok.line, p.tok.value)
	}
	keyword := p.tok.value
	switch keyword {
	case "global":
		p.lastCmd = nil
		p.lastTopic = nil
		p.next()
		return nil
	case "cmd":
		return p.parseCommand()
	case "flag":
		return p.parseFlag()
	case "arg":
		return p.parseArg()
	case "param":
		return p.parseParam()
	case "name":
		return p.parseName()
	case "example":
		return p.parseExample()
	case "topic":
		return p.parseTopic()
	case "text":
		return p.parseText()
	default:
		return fmt.Errorf("line %d: unknown keyword %q", p.tok.line, keyword)
	}
}

func (p *parser) parseFlag() error {
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected flag name", p.tok.line)
	}
	name := p.tok.value
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected flag type", p.tok.line)
	}
	fType := p.tok.value
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected flag description", p.tok.line)
	}
	desc := p.tok.value
	p.next()
	f := flag{Name: name, Type: fType, Desc: desc}
	if p.tok.kind == tokIdentifier {
		f.Short = p.tok.value
		p.next()
	}
	if p.lastCmd == nil {
		p.def.GlobalFlags = append(p.def.GlobalFlags, f)
	} else {
		p.lastCmd.Flags = append(p.lastCmd.Flags, f)
	}
	return nil
}

func (p *parser) parseCommand() error {
	p.next()
	path := p.pathBuf[:0]
	for p.tok.kind == tokIdentifier {
		path = append(path, p.tok.value)
		p.next()
	}
	if len(path) == 0 {
		return fmt.Errorf("line %d: expected command name or path", p.tok.line)
	}
	desc := ""
	if p.tok.kind == tokString {
		desc = p.tok.value
		p.next()
	}
	var parent *command
	var current *command
	for i, name := range path {
		var list *[]*command
		if parent == nil {
			list = &p.def.Commands
		} else {
			list = &parent.Subs
		}
		found := false
		for _, c := range *list {
			if c.Name == name {
				current = c
				found = true
				break
			}
		}
		if !found {
			current = &command{Name: name, Parent: parent}
			*list = append(*list, current)
		}
		if i == len(path)-1 && desc != "" {
			current.Desc = desc
		}
		parent = current
	}
	p.lastCmd = current
	return nil
}

func (p *parser) parseArg() error {
	if p.lastCmd == nil {
		return fmt.Errorf("line %d: 'arg' must follow a 'cmd'", p.tok.line)
	}
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected arg name", p.tok.line)
	}
	name := p.tok.value
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected arg type", p.tok.line)
	}
	aType := p.tok.value
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected arg description", p.tok.line)
	}
	desc := p.tok.value
	p.next()
	p.lastCmd.Args = append(p.lastCmd.Args, arg{Name: name, Type: aType, Desc: desc})
	return nil
}

func (p *parser) parseParam() error {
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected param name", p.tok.line)
	}
	name := p.tok.value
	p.next()
	if p.tok.kind != tokEquals {
		return fmt.Errorf("line %d: expected '=' after param name", p.tok.line)
	}
	p.next()

	var val value
	switch p.tok.kind {
	case tokString:
		val = value{Kind: "string", Str: p.tok.value}
	case tokIdentifier:
		if p.tok.value == "true" || p.tok.value == "false" {
			val = value{Kind: "bool", Bool: p.tok.value == "true"}
		} else {
			return fmt.Errorf("line %d: expected bool or string param value", p.tok.line)
		}
	case tokNumber:
		n, err := strconv.Atoi(p.tok.value)
		if err != nil {
			return fmt.Errorf("line %d: invalid number %q", p.tok.line, p.tok.value)
		}
		val = value{Kind: "int", Int: n}
	default:
		return fmt.Errorf("line %d: expected param value", p.tok.line)
	}
	p.next()

	if p.lastCmd == nil {
		if p.def.GlobalParams == nil {
			p.def.GlobalParams = map[string]value{}
		}
		p.def.GlobalParams[name] = val
		return nil
	}
	if p.lastCmd.Params == nil {
		p.lastCmd.Params = map[string]value{}
	}
	p.lastCmd.Params[name] = val
	return nil
}

func (p *parser) parseName() error {
	if p.lastCmd != nil {
		return fmt.Errorf("line %d: 'name' must be under 'global'", p.tok.line)
	}
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected binary name string", p.tok.line)
	}
	p.def.AppName = p.tok.value
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected tagline string", p.tok.line)
	}
	p.def.Tagline = p.tok.value
	p.next()
	return nil
}

func (p *parser) parseExample() error {
	if p.lastCmd == nil {
		return fmt.Errorf("line %d: 'example' must follow a 'cmd'", p.tok.line)
	}
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected example string", p.tok.line)
	}
	p.lastCmd.Examples = append(p.lastCmd.Examples, p.tok.value)
	p.next()
	return nil
}

func (p *parser) parseTopic() error {
	p.next()
	if p.tok.kind != tokIdentifier {
		return fmt.Errorf("line %d: expected topic name", p.tok.line)
	}
	name := p.tok.value
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected topic description", p.tok.line)
	}
	desc := p.tok.value
	p.next()
	t := topic{Name: name, Desc: desc}
	p.def.Topics = append(p.def.Topics, t)
	p.lastTopic = &p.def.Topics[len(p.def.Topics)-1]
	return nil
}

func (p *parser) parseText() error {
	if p.lastTopic == nil {
		return fmt.Errorf("line %d: 'text' must follow a 'topic'", p.tok.line)
	}
	p.next()
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected text string", p.tok.line)
	}
	p.lastTopic.Text = p.tok.value
	p.next()
	return nil
}
