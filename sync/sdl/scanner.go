// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package sdl

import "unicode"

// scanner walks one SDL source file rune by rune.
//
// Deliberately not a tokeniser: the grammar accepted here is small enough
// that the parsers below read structure directly, and a token stream would
// add a layer without removing a case.
type scanner struct {
	src  []rune
	pos  int
	file string
}

// newScanner starts a scanner over src.
func newScanner(file, src string) *scanner {
	return &scanner{src: []rune(src), file: file}
}

func (s *scanner) eof() bool { return s.pos >= len(s.src) }

// peek returns the rune at the cursor, or 0 at end of input.
func (s *scanner) peek() rune {
	if s.eof() {
		return 0
	}
	return s.src[s.pos]
}

// next returns the rune at the cursor and advances past it.
func (s *scanner) next() rune {
	r := s.peek()
	s.pos++
	return r
}

// skipIgnored advances past whitespace, commas (which GraphQL treats as
// whitespace) and `#` comments.
func (s *scanner) skipIgnored() {
	for !s.eof() {
		switch r := s.peek(); {
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

// hasPrefix reports whether the input at the cursor begins with p.
func (s *scanner) hasPrefix(p string) bool {
	end := s.pos + len([]rune(p))
	if end > len(s.src) {
		return false
	}
	return string(s.src[s.pos:end]) == p
}

// delimited consumes a body bounded by open/close, returning its inner text
// with the cursor left after the closer. Nested pairs are counted, and a
// description inside the body is skipped whole so a brace or paren in prose
// cannot unbalance the count.
func (s *scanner) delimited(open, close rune) string {
	if s.peek() != open {
		return ""
	}
	s.pos++
	depth, start := 1, s.pos
	for !s.eof() {
		switch s.peek() {
		case open:
			depth++
		case close:
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

// block consumes a brace-delimited body and returns its inner text.
func (s *scanner) block() string {
	s.skipIgnored()
	return s.delimited('{', '}')
}

// argList consumes a parenthesised argument list and returns its inner text.
func (s *scanner) argList() string { return s.delimited('(', ')') }
