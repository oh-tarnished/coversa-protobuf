// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package sdl

// description.go reads GraphQL description strings and normalises them into
// the comments the emitted protobuf carries.
//
// Kept apart from the scanner primitives because descriptions are the one
// place the two forms of quoting matter -- a `"""block"""` may contain braces
// and quotes that would otherwise unbalance the delimiter counting.

import "strings"

// description reads a leading `"""block"""` or `"single line"` documentation
// string, returning "" when the cursor is not on one.
func (s *scanner) description() string {
	s.skipIgnored()
	if s.peek() != '"' {
		return ""
	}
	if s.hasPrefix(`"""`) {
		return s.blockDescription()
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

// blockDescription reads a `"""..."""` description, cursor on the opener.
func (s *scanner) blockDescription() string {
	s.pos += 3
	start := s.pos
	for !s.eof() {
		if s.peek() == '"' && s.hasPrefix(`"""`) {
			text := string(s.src[start:s.pos])
			s.pos += 3
			return cleanDoc(text)
		}
		s.pos++
	}
	return cleanDoc(string(s.src[start:]))
}

// cleanDoc normalises a description block: tabs to spaces, the common
// indentation removed, and surrounding blank lines dropped.
func cleanDoc(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\t", "    "), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	indent := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if n := len(l) - len(strings.TrimLeft(l, " ")); indent < 0 || n < indent {
			indent = n
		}
	}
	for i, l := range lines {
		if indent > 0 && len(l) >= indent {
			l = l[indent:]
		}
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}
