// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// sdl.go parses the subset of GraphQL SDL that the VDM specification uses.
//
// A general GraphQL parser is not needed and is not wanted: the input is a
// generated, highly regular corpus, and a hand-written scanner over it fails
// loudly on anything it has not seen rather than silently accepting a shape
// the emitter downstream cannot render. Every construct below appears in
// vdm/spec; nothing is supported speculatively.
//
// Deliberately stdlib-only, so `go run tools/vssgen/*.go` works with no module
// context and no dependency to keep current.

import (
	"fmt"
	"strings"
)

// Arg is one argument supplied to a directive, e.g. `element: SENSOR`.
type Arg struct {
	Name string // argument name
	Raw  string // literal text of the value, quotes already stripped
	Str  bool   // true when the value was a quoted string
}

// Directive is one `@name(...)` annotation.
type Directive struct {
	Name string
	Args []Arg
}

// Arg returns the named argument's raw value and whether it was present.
func (d Directive) Arg(name string) (string, bool) {
	for _, a := range d.Args {
		if a.Name == name {
			return a.Raw, true
		}
	}
	return "", false
}

// TypeRef is a reference to a type in a field or argument position, carrying
// the list and non-null decoration GraphQL wraps around the bare name.
type TypeRef struct {
	Name        string // the underlying named type
	List        bool   // the reference is [T]
	NonNull     bool   // the reference itself is T!
	ItemNonNull bool   // list items are [T!]
}

// ArgDef is a field argument declaration, e.g. `unit: VelocityUnitEnum = METER`.
//
// The VDM spec uses these for one purpose only -- naming the unit a signal's
// value is expressed in, with the canonical unit as the default -- and that
// default is what this generator pins into the schema.
type ArgDef struct {
	Name    string
	Type    TypeRef
	Default string
}

// Field is one field of an object type.
type Field struct {
	Name       string
	Doc        string
	Type       TypeRef
	Args       []ArgDef
	Directives []Directive
}

// EnumValue is one member of an enum type.
type EnumValue struct {
	Name       string
	Doc        string
	Directives []Directive
}

// Def is a top-level definition: an object type or an enum.
type Def struct {
	Kind       string // "type", "enum", "extend type", "scalar", "directive"
	Name       string
	Doc        string
	Directives []Directive
	Fields     []Field
	Values     []EnumValue
	File       string
}

// Directive returns the named directive on this definition, if present.
func (d Def) Directive(name string) (Directive, bool) {
	for _, x := range d.Directives {
		if x.Name == name {
			return x, true
		}
	}
	return Directive{}, false
}

// Directive returns the named directive on this field, if present.
func (f Field) Directive(name string) (Directive, bool) {
	for _, x := range f.Directives {
		if x.Name == name {
			return x, true
		}
	}
	return Directive{}, false
}

// Directive returns the named directive on this enum value, if present.
func (v EnumValue) Directive(name string) (Directive, bool) {
	for _, x := range v.Directives {
		if x.Name == name {
			return x, true
		}
	}
	return Directive{}, false
}

// scanner walks one SDL source file.
type scanner struct {
	src  []rune
	pos  int
	file string
}

// ParseSDL parses one SDL document into its top-level definitions.
func ParseSDL(file, src string) ([]Def, error) {
	s := &scanner{src: []rune(src), file: file}
	var defs []Def
	for {
		s.skipIgnored()
		if s.eof() {
			return defs, nil
		}
		doc := s.description()
		s.skipIgnored()
		if s.eof() {
			return defs, nil
		}
		kw := s.word()
		if kw == "" {
			return nil, fmt.Errorf("%s: expected a keyword at offset %d", file, s.pos)
		}
		def := Def{Kind: kw, Doc: doc, File: file}
		if kw == "extend" {
			s.skipIgnored()
			def.Kind = "extend " + s.word()
		}
		s.skipIgnored()
		def.Name = s.word()
		// `schema { ... }` has no name; it is not modelled and is skipped whole.
		if def.Kind == "schema" {
			s.block()
			continue
		}
		// A directive *definition* is consumed and discarded. This generator
		// interprets the three directives the spec uses -- @vspec, @range and
		// @instanceTag -- directly, so their declarations carry nothing it needs.
		if def.Kind == "directive" {
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
			if s.peekWord() == "on" {
				s.word()
				for {
					s.skipIgnored()
					if s.word() == "" {
						break
					}
					s.skipIgnored()
					if s.peek() != '|' {
						break
					}
					s.next()
				}
			}
			defs = append(defs, def)
			continue
		}

		def.Directives = s.directives()
		s.skipIgnored()
		switch {
		case s.peek() == '{' && strings.HasSuffix(def.Kind, "type"):
			body := s.block()
			fs, err := parseFields(file, body)
			if err != nil {
				return nil, err
			}
			def.Fields = fs
		case s.peek() == '{' && def.Kind == "enum":
			body := s.block()
			vs, err := parseEnumValues(file, body)
			if err != nil {
				return nil, err
			}
			def.Values = vs
		}
		defs = append(defs, def)
	}
}
