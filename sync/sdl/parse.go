// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package sdl

import "fmt"

// Parse reads one SDL document into its top-level definitions.
func Parse(file, src string) ([]Def, error) {
	s := newScanner(file, src)
	var defs []Def
	for {
		s.skipIgnored()
		if s.eof() {
			return defs, nil
		}
		def, ok, err := s.definition()
		if err != nil {
			return nil, err
		}
		if !ok {
			return defs, nil
		}
		if def.Kind != "" {
			defs = append(defs, def)
		}
	}
}

// definition reads one top-level definition.
//
// The second result is false at end of input. A definition the generator does
// not model -- `schema { ... }` -- is consumed and returned with an empty
// Kind, so the caller drops it without treating it as an error.
func (s *scanner) definition() (Def, bool, error) {
	doc := s.description()
	s.skipIgnored()
	if s.eof() {
		return Def{}, false, nil
	}

	kw := s.word()
	if kw == "" {
		return Def{}, false, fmt.Errorf("%s: expected a keyword at offset %d", s.file, s.pos)
	}

	def := Def{Kind: kw, Doc: doc, File: s.file}
	if kw == "extend" {
		s.skipIgnored()
		def.Kind = "extend " + s.word()
	}
	s.skipIgnored()
	def.Name = s.word()

	switch {
	case def.Kind == "schema":
		// Names the query root and declares no data. Consumed and dropped.
		s.block()
		return Def{}, true, nil
	case def.Kind == "directive":
		s.directiveDefinition(&def)
		return def, true, nil
	}

	def.Directives = s.directives()
	s.skipIgnored()
	if s.peek() != '{' {
		return def, true, nil
	}
	return def, true, s.body(&def)
}

// body parses the brace-delimited members of an object type or enum.
func (s *scanner) body(def *Def) error {
	body := s.block()
	switch {
	case isTypeKind(def.Kind):
		fields, err := parseFields(s.file, body)
		if err != nil {
			return err
		}
		def.Fields = fields
	case def.Kind == "enum":
		values, err := parseEnumValues(s.file, body)
		if err != nil {
			return err
		}
		def.Values = values
	}
	return nil
}

// isTypeKind reports whether a definition kind carries object fields.
func isTypeKind(kind string) bool {
	return kind == "type" || kind == "extend type"
}

// directiveDefinition consumes a `directive @x(...) on LOC | LOC` declaration.
//
// Recorded by name and otherwise discarded: the generator interprets the
// directives the specification uses -- @vspec, @range, @instanceTag,
// @deprecated -- directly, so their declarations carry nothing it needs.
func (s *scanner) directiveDefinition(def *Def) {
	s.skipIgnored()
	if s.peek() == '@' {
		s.next()
	}
	def.Name = s.word()

	s.skipIgnored()
	if s.peek() == '(' {
		s.argList()
	}
	s.skipIgnored()
	if s.peekWord() != "on" {
		return
	}

	s.word()
	for {
		s.skipIgnored()
		if s.word() == "" {
			return
		}
		s.skipIgnored()
		if s.peek() != '|' {
			return
		}
		s.next()
	}
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
