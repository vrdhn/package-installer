package main

import (
	"fmt"
	"strings"
)

type tokenKind int

const (
	tokError tokenKind = iota
	tokEOF
	tokIdentifier
	tokString
	tokEquals
	tokNumber
)

type token struct {
	kind  tokenKind
	value string
	line  int
}

type lexer struct {
	input string
	pos   int
	line  int
}

func newLexer(input string) *lexer {
	return &lexer{
		input: input,
		line:  1,
	}
}

func (l *lexer) nextToken() token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return token{kind: tokEOF, line: l.line}
	}

	b := l.input[l.pos]

	if b == '"' {
		return l.readString()
	}

	if b == '=' {
		l.pos++
		return token{kind: tokEquals, value: "=", line: l.line}
	}

	if b >= '0' && b <= '9' {
		return l.readNumber()
	}

	if isAlpha(b) || b == '/' || b == '.' {
		return l.readIdentifier()
	}

	l.pos++
	return token{kind: tokError, value: fmt.Sprintf("unexpected character: %c", b), line: l.line}
}

func (l *lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		b := l.input[l.pos]
		switch b {
		case '\n':
			l.line++
			l.pos++
		case ' ', '\t', '\r', '\f', '\v':
			l.pos++
		case '#':
			l.skipComment()
		default:
			return
		}
	}
}

func (l *lexer) skipComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *lexer) readIdentifier() token {
	start := l.pos
	for l.pos < len(l.input) {
		b := l.input[l.pos]
		if isAlphaNumeric(b) || b == '_' || b == '-' || b == '/' || b == '.' {
			l.pos++
		} else {
			break
		}
	}
	return token{kind: tokIdentifier, value: l.input[start:l.pos], line: l.line}
}

func (l *lexer) readNumber() token {
	start := l.pos
	for l.pos < len(l.input) {
		b := l.input[l.pos]
		if b >= '0' && b <= '9' {
			l.pos++
		} else {
			break
		}
	}
	return token{kind: tokNumber, value: l.input[start:l.pos], line: l.line}
}

func (l *lexer) readString() token {
	l.pos++
	if l.pos+1 < len(l.input) && l.input[l.pos] == '"' && l.input[l.pos+1] == '"' {
		l.pos += 2
		return l.readMultilineString()
	}
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\n' {
			l.line++
		}
		l.pos++
	}
	if l.pos >= len(l.input) {
		return token{kind: tokError, value: "unterminated string", line: l.line}
	}
	val := l.input[start:l.pos]
	l.pos++
	return token{kind: tokString, value: val, line: l.line}
}

func (l *lexer) readMultilineString() token {
	start := l.pos
	for l.pos+2 < len(l.input) {
		if l.input[l.pos] == '"' && l.input[l.pos+1] == '"' && l.input[l.pos+2] == '"' {
			val := l.input[start:l.pos]
			l.pos += 3
			return token{kind: tokString, value: l.processMultiline(val), line: l.line}
		}
		if l.input[l.pos] == '\n' {
			l.line++
		}
		l.pos++
	}
	return token{kind: tokError, value: "unterminated multiline string", line: l.line}
}

func (l *lexer) processMultiline(s string) string {
	var sb strings.Builder
	lines := strings.Split(s, "\n")
	startIdx := 0
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}
	endIdx := len(lines) - 1
	for endIdx >= startIdx && strings.TrimSpace(lines[endIdx]) == "" {
		endIdx--
	}
	if startIdx > endIdx {
		return ""
	}
	for i := startIdx; i <= endIdx; i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed != "" {
			sb.WriteByte(' ')
			sb.WriteString(trimmed)
		}
		if i < endIdx {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isAlphaNumeric(b byte) bool {
	return isAlpha(b) || (b >= '0' && b <= '9')
}
