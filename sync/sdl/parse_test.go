// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package sdl

import "testing"

// TestParseBranch covers the shape most of the specification is made of: a
// documented object type carrying a @vspec directive, fields with unit
// arguments, a @range bound and a list-valued branch reference.
func TestParseBranch(t *testing.T) {
	const src = `
"""All in-cabin components."""
type Cabin @vspec(element: BRANCH, fqn: "Vehicle.Cabin") {
  """Number of doors."""
  doorCount: UInt8 @vspec(element: ATTRIBUTE, fqn: "Vehicle.Cabin.DoorCount")

  """Power optimization level."""
  powerOptimizeLevel: UInt8 @vspec(element: ACTUATOR, fqn: "Vehicle.Cabin.Level")
    @range(min: 0, max: 10)

  seats: [Seat] @vspec(element: BRANCH, fqn: "Vehicle.Cabin.Seat")
  speed(unit: VelocityUnitEnum = KILOMETER_PER_HOUR): Float
}
`
	defs, err := Parse("cabin.graphql", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}

	d := defs[0]
	if d.Kind != "type" || d.Name != "Cabin" {
		t.Errorf("got %s %q, want type Cabin", d.Kind, d.Name)
	}
	if d.Doc != "All in-cabin components." {
		t.Errorf("doc = %q", d.Doc)
	}
	if v, ok := d.Directive("vspec"); !ok {
		t.Error("no @vspec on the type")
	} else if fqn, _ := v.Arg("fqn"); fqn != "Vehicle.Cabin" {
		t.Errorf("fqn = %q", fqn)
	}
	if len(d.Fields) != 4 {
		t.Fatalf("got %d fields, want 4", len(d.Fields))
	}

	// A @range directive alongside a @vspec on the same field.
	level := d.Fields[1]
	r, ok := level.Directive("range")
	if !ok {
		t.Fatal("no @range on powerOptimizeLevel")
	}
	if lo, _ := r.Arg("min"); lo != "0" {
		t.Errorf("min = %q, want 0", lo)
	}
	if hi, _ := r.Arg("max"); hi != "10" {
		t.Errorf("max = %q, want 10", hi)
	}

	// A list-valued branch, which is what marks members as identifiable.
	if seats := d.Fields[2]; !seats.Type.List || seats.Type.Name != "Seat" {
		t.Errorf("seats type = %+v, want list of Seat", seats.Type)
	}

	// The unit argument and its default, which the schema pins.
	speed := d.Fields[3]
	if len(speed.Args) != 1 {
		t.Fatalf("got %d args on speed, want 1", len(speed.Args))
	}
	if a := speed.Args[0]; a.Name != "unit" || a.Default != "KILOMETER_PER_HOUR" {
		t.Errorf("speed arg = %+v", a)
	}
}

// TestParseEnumAndExtend covers the other two shapes: an enum whose values
// carry an originalName, and the `extend type` the instance-tag files use.
func TestParseEnumAndExtend(t *testing.T) {
	const src = `
"""Dimensional enum."""
enum Seat_InstanceTag_Dimension1 {
  ROW1 @vspec(originalName: "Row1")
  ROW2 @vspec(originalName: "Row2")
}

extend type Seat {
  instanceTag: Seat_InstanceTag
}
`
	defs, err := Parse("seat.graphql", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("got %d definitions, want 2", len(defs))
	}

	e := defs[0]
	if e.Kind != "enum" || len(e.Values) != 2 {
		t.Fatalf("got %s with %d values", e.Kind, len(e.Values))
	}
	v, ok := e.Values[0].Directive("vspec")
	if !ok {
		t.Fatal("no @vspec on ROW1")
	}
	if orig, _ := v.Arg("originalName"); orig != "Row1" {
		t.Errorf("originalName = %q, want Row1", orig)
	}

	if x := defs[1]; x.Kind != "extend type" || x.Name != "Seat" {
		t.Errorf("got %s %q, want extend type Seat", x.Kind, x.Name)
	}
}

// TestParseRejectsUnknownShape is the rule the package exists to enforce: an
// unfamiliar construct is an error, never a silently skipped field.
func TestParseRejectsUnknownShape(t *testing.T) {
	const src = `type Broken { fieldWithNoType }`
	if _, err := Parse("broken.graphql", src); err == nil {
		t.Fatal("Parse accepted a field with no type; want an error")
	}
}

// TestCleanDoc checks the indentation stripping that makes a block
// description usable as a protobuf comment.
func TestCleanDoc(t *testing.T) {
	got := cleanDoc("\n    first line\n      indented\n\n    last\n  ")
	want := "first line\n  indented\n\nlast"
	if got != want {
		t.Errorf("cleanDoc =\n%q\nwant\n%q", got, want)
	}
}
