// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// parse.go parses the two bodies a definition can have -- an object type's
// fields and an enum's values -- plus the argument lists and type references
// inside them.
import (
	"fmt"
	"strings"
	"unicode"
)

// parseArgs splits a directive argument list into name/value pairs.
func parseArgs(body string) []Arg {
	s := &scanner{src: []rune(body)}
	var out []Arg
	for {
		s.skipIgnored()
		if s.eof() {
			return out
		}
		name := s.word()
		if name == "" {
			return out
		}
		s.skipIgnored()
		if s.peek() != ':' {
			continue
		}
		s.pos++
		s.skipIgnored()
		if s.peek() == '"' {
			out = append(out, Arg{Name: name, Raw: s.description(), Str: true})
			continue
		}
		start := s.pos
		for !s.eof() && !unicode.IsSpace(s.peek()) && s.peek() != ',' {
			s.pos++
		}
		out = append(out, Arg{Name: name, Raw: strings.TrimSpace(string(s.src[start:s.pos]))})
	}
}

// typeRef reads a type reference, unwrapping list and non-null decoration.
func (s *scanner) typeRef() TypeRef {
	s.skipIgnored()
	var t TypeRef
	if s.peek() == '[' {
		t.List = true
		s.pos++
		s.skipIgnored()
		t.Name = s.word()
		if s.peek() == '!' {
			t.ItemNonNull = true
			s.pos++
		}
		s.skipIgnored()
		if s.peek() == ']' {
			s.pos++
		}
	} else {
		t.Name = s.word()
	}
	if s.peek() == '!' {
		t.NonNull = true
		s.pos++
	}
	return t
}

// parseFields parses an object type body into its fields.
func parseFields(file, body string) ([]Field, error) {
	s := &scanner{src: []rune(body), file: file}
	var out []Field
	for {
		doc := s.description()
		s.skipIgnored()
		if s.eof() {
			return out, nil
		}
		f := Field{Doc: doc, Name: s.word()}
		if f.Name == "" {
			return nil, fmt.Errorf("%s: expected a field name at offset %d", file, s.pos)
		}
		s.skipIgnored()
		if s.peek() == '(' {
			f.Args = parseArgDefs(s.argList())
		}
		s.skipIgnored()
		if s.peek() != ':' {
			return nil, fmt.Errorf("%s: field %q has no type", file, f.Name)
		}
		s.pos++
		f.Type = s.typeRef()
		f.Directives = s.directives()
		out = append(out, f)
	}
}

// parseArgDefs parses a field's argument declarations, including defaults.
func parseArgDefs(body string) []ArgDef {
	s := &scanner{src: []rune(body)}
	var out []ArgDef
	for {
		s.description()
		s.skipIgnored()
		if s.eof() {
			return out
		}
		a := ArgDef{Name: s.word()}
		if a.Name == "" {
			return out
		}
		s.skipIgnored()
		if s.peek() != ':' {
			continue
		}
		s.pos++
		a.Type = s.typeRef()
		s.skipIgnored()
		if s.peek() == '=' {
			s.pos++
			s.skipIgnored()
			a.Default = s.word()
		}
		out = append(out, a)
	}
}

// parseEnumValues parses an enum body into its values.
func parseEnumValues(file, body string) ([]EnumValue, error) {
	s := &scanner{src: []rune(body), file: file}
	var out []EnumValue
	for {
		doc := s.description()
		s.skipIgnored()
		if s.eof() {
			return out, nil
		}
		v := EnumValue{Doc: doc, Name: s.word()}
		if v.Name == "" {
			return nil, fmt.Errorf("%s: expected an enum value at offset %d", file, s.pos)
		}
		v.Directives = s.directives()
		out = append(out, v)
	}
}

// cleanDoc normalises a description block: tabs to spaces, common indentation
// removed, surrounding blank lines dropped.
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
		n := len(l) - len(strings.TrimLeft(l, " "))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i, l := range lines {
		if len(l) >= indent && indent > 0 {
			lines[i] = l[indent:]
		}
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}
