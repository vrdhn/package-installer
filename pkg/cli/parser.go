package cli

import (
	"fmt"
)

// Mutable
type parser struct {
	lex       *lexer
	tok       token
	engine    *Engine
	lastCmd   *Command
	lastTopic *Topic
	pathBuf   [8]string // Reusable buffer for short paths to avoid allocations
}

func newParser(dsl string, engine *Engine) *parser {
	p := &parser{
		lex:    newLexer(dsl),
		engine: engine,
	}
	p.next()
	return p
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
	case "example":
		return p.parseExample()
	case "topic":
		return p.parseTopic()
	case "text":
		return p.parseText()
	case "safe":
		if p.lastCmd == nil {
			return fmt.Errorf("line %d: 'safe' must follow a 'cmd'", p.tok.line)
		}
		p.lastCmd.SafeInCave = true
		p.next()
		return nil
	default:
		panic("unreachable")
	}
}

func (p *parser) parseFlag() error {
	p.next() // skip 'flag'
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

	f := &Flag{Name: name, Type: fType, Desc: desc}

	if p.tok.kind == tokIdentifier {
		f.Short = p.tok.value
		p.next()
	}

	if p.lastCmd == nil {
		p.engine.GlobalFlags = append(p.engine.GlobalFlags, f)
	} else {
		p.lastCmd.Flags = append(p.lastCmd.Flags, f)
	}
	return nil
}

func (p *parser) parseCommand() error {
	p.next() // skip 'cmd'

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

	var parent *Command
	var current *Command

	for i, name := range path {
		var list *[]*Command
		if parent == nil {
			list = &p.engine.Commands
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
			current = &Command{Name: name, Parent: parent}
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
	p.next() // skip 'arg'

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

	p.lastCmd.Args = append(p.lastCmd.Args, &Arg{Name: name, Type: aType, Desc: desc})
	return nil
}

func (p *parser) parseExample() error {
	if p.lastCmd == nil {
		return fmt.Errorf("line %d: 'example' must follow a 'cmd'", p.tok.line)
	}
	p.next() // skip 'example'
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected example string", p.tok.line)
	}
	p.lastCmd.Examples = append(p.lastCmd.Examples, p.tok.value)
	p.next()
	return nil
}

func (p *parser) parseTopic() error {
	p.next() // skip 'topic'
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

	t := &Topic{Name: name, Desc: desc}
	p.engine.Topics = append(p.engine.Topics, t)
	p.lastTopic = t
	return nil
}

func (p *parser) parseText() error {
	if p.lastTopic == nil {
		return fmt.Errorf("line %d: 'text' must follow a 'topic'", p.tok.line)
	}
	p.next() // skip 'text'
	if p.tok.kind != tokString {
		return fmt.Errorf("line %d: expected text string", p.tok.line)
	}
	p.lastTopic.Text = p.tok.value
	p.next()
	return nil
}
