// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

// Package sdl parses the subset of GraphQL SDL that the COVESA Vehicle Data
// Model is written in.
//
// # Why not a general GraphQL parser
//
// A full parser is not needed and is not wanted. The input is a generated,
// highly regular corpus, and a hand-written scanner over it fails loudly on
// anything it has not seen rather than silently accepting a shape the
// emitters downstream cannot render. Every construct supported here appears
// in vdm/spec; nothing is supported speculatively.
//
// That is the rule the whole generator is built on: an unknown construct is
// an error, never a skipped field. A parser that quietly accepts an
// unfamiliar shape produces a schema that is wrong in a way no linter
// catches.
package sdl

// Arg is one argument supplied to a directive, e.g. `element: SENSOR`.
type Arg struct {
	Name string // argument name
	Raw  string // literal text of the value, quotes already stripped
	Str  bool   // the value was a quoted string
}

// Directive is one `@name(...)` annotation.
//
// The specification uses four: @vspec carries the VSS element kind and fully
// qualified name, @range carries numeric bounds, @instanceTag marks a branch
// whose members have identity, and @deprecated carries a reason.
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

// ArgDef is a field argument declaration, e.g.
// `unit: VelocityUnitEnum = KILOMETER_PER_HOUR`.
//
// The specification uses these for one purpose: naming the unit a signal's
// value is expressed in, with the canonical unit as the default. Protobuf has
// no field arguments, so that default is what the generator pins into the
// schema.
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

// Directive returns the named directive on this field, if present.
func (f Field) Directive(name string) (Directive, bool) {
	return findDirective(f.Directives, name)
}

// EnumValue is one member of an enum type.
type EnumValue struct {
	Name       string
	Doc        string
	Directives []Directive
}

// Directive returns the named directive on this enum value, if present.
func (v EnumValue) Directive(name string) (Directive, bool) {
	return findDirective(v.Directives, name)
}

// Def is a top-level definition: an object type, an enum, a scalar or a
// directive declaration.
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
	return findDirective(d.Directives, name)
}

// findDirective is the shared lookup behind the three Directive methods.
func findDirective(ds []Directive, name string) (Directive, bool) {
	for _, x := range ds {
		if x.Name == name {
			return x, true
		}
	}
	return Directive{}, false
}
