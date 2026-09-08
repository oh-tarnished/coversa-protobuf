// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// scan.go holds the scanner primitives ParseSDL is built from: whitespace and
// comment skipping, identifier and description reading, brace matching, and
// the two body parsers for object fields and enum values.

import (
	"strings"
	"unicode"
)

func (s *scanner) eof() bool { return s.pos >= len(s.src) }

func (s *scanner) peek() rune {
	if s.eof() {
		return 0
	}
	return s.src[s.pos]
}

func (s *scanner) next() rune {
	r := s.peek()
	s.pos++
	return r
}

// skipIgnored advances past whitespace, commas (which GraphQL treats as
// whitespace) and `#` comments.
func (s *scanner) skipIgnored() {
	for !s.eof() {
		r := s.peek()
		switch {
		case unicode.IsSpace(r) || r == ',':
			s.pos++
		case r == '#':
			s.until('\n')
		default:
			return
		}
	}
}

// until consumes runes through the first occurrence of stop.
func (s *scanner) until(stop rune) {
	for !s.eof() && s.next() != stop {
	}
}

// isNameRune reports whether r may appear in a GraphQL name.
func isNameRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// word reads one name, returning "" when the cursor is not on a name.
func (s *scanner) word() string {
	start := s.pos
	for !s.eof() && isNameRune(s.peek()) {
		s.pos++
	}
	return string(s.src[start:s.pos])
}

// peekWord reads the next name without advancing.
func (s *scanner) peekWord() string {
	save := s.pos
	w := s.word()
	s.pos = save
	return w
}

// description reads a leading `"""block"""` or `"single line"` documentation
// string, returning "" when the cursor is not on one.
func (s *scanner) description() string {
	s.skipIgnored()
	if s.peek() != '"' {
		return ""
	}
	if strings.HasPrefix(string(s.src[s.pos:min(s.pos+3, len(s.src))]), `"""`) {
		s.pos += 3
		start := s.pos
		for !s.eof() {
			if s.peek() == '"' && strings.HasPrefix(string(s.src[s.pos:min(s.pos+3, len(s.src))]), `"""`) {
				text := string(s.src[start:s.pos])
				s.pos += 3
				return cleanDoc(text)
			}
			s.pos++
		}
		return cleanDoc(string(s.src[start:]))
	}
	s.pos++ // opening quote
	start := s.pos
	for !s.eof() && s.peek() != '"' {
		s.pos++
	}
	text := string(s.src[start:s.pos])
	s.pos++ // closing quote
	return cleanDoc(text)
}

// block consumes a brace-delimited body and returns its inner text.
func (s *scanner) block() string {
	s.skipIgnored()
	if s.peek() != '{' {
		return ""
	}
	s.pos++
	depth, start := 1, s.pos
	for !s.eof() {
		switch s.peek() {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				body := string(s.src[start:s.pos])
				s.pos++
				return body
			}
		case '"':
			// A description inside the body may itself contain braces.
			save := s.pos
			s.description()
			if s.pos == save {
				s.pos++
			}
			continue
		}
		s.pos++
	}
	return string(s.src[start:])
}

// directives reads a run of `@name(args)` annotations.
func (s *scanner) directives() []Directive {
	var out []Directive
	for {
		s.skipIgnored()
		if s.peek() != '@' {
			return out
		}
		s.pos++
		d := Directive{Name: s.word()}
		s.skipIgnored()
		if s.peek() == '(' {
			d.Args = parseArgs(s.argList())
		}
		out = append(out, d)
	}
}

// argList consumes a parenthesised argument list and returns its inner text.
func (s *scanner) argList() string {
	s.pos++ // '('
	depth, start := 1, s.pos
	for !s.eof() {
		switch s.peek() {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				body := string(s.src[start:s.pos])
				s.pos++
				return body
			}
		case '"':
			save := s.pos
			s.description()
			if s.pos == save {
				s.pos++
			}
			continue
		}
		s.pos++
	}
	return string(s.src[start:])
}
