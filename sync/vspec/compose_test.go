// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vspec

// compose_test.go covers how a specification is assembled from many files:
// the `#include` preprocessor, dotted keys, instance expansion, and the two
// ways composition is allowed to fail.

import (
	"path/filepath"
	"testing"
)

// TestIncludeAndDottedKeys covers VSS's two composition mechanisms: the
// `#include` preprocessor, and a key that is itself a dotted path.
func TestIncludeAndDottedKeys(t *testing.T) {
	dir := write(t, map[string]string{
		"root.vspec": `
Vehicle:
  type: branch
  description: Root.

Vehicle.Cabin:
  type: branch
  description: The cabin.
#include Cabin/Seat.vspec Vehicle.Cabin
`,
		"Cabin/Seat.vspec": `
Seat:
  type: branch
  description: All seats.
  instances:
    - Row[1,2]
    - ["DriverSide", "Middle", "PassengerSide"]
  IsBelted:
    datatype: boolean
    type: sensor
    description: Is the belt engaged.
`,
	})
	root, err := Parse(filepath.Join(dir, "root.vspec"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// `Vehicle.Cabin:` declares Cabin *beneath* Vehicle, not a sibling.
	cabin, ok := root.Child("Cabin")
	if !ok {
		t.Fatal("Cabin was not spliced under Vehicle")
	}
	seat, ok := cabin.Child("Seat")
	if !ok {
		t.Fatal("the include did not attach Seat")
	}
	if seat.FQN != "Vehicle.Cabin.Seat" {
		t.Errorf("seat fqn = %q", seat.FQN)
	}
	if belted, ok := seat.Child("IsBelted"); !ok || belted.FQN != "Vehicle.Cabin.Seat.IsBelted" {
		t.Errorf("nested fqn wrong: %+v", belted)
	}

	// Two axes, the first expanded from Row[1,2].
	want := [][]string{{"Row1", "Row2"}, {"DriverSide", "Middle", "PassengerSide"}}
	if len(seat.Instances) != 2 {
		t.Fatalf("instances = %v, want two axes", seat.Instances)
	}
	for i, axis := range want {
		for j, name := range axis {
			if seat.Instances[i][j] != name {
				t.Errorf("instances[%d][%d] = %q, want %q", i, j, seat.Instances[i][j], name)
			}
		}
	}
}

// TestNumberedEnum covers the form the GraphQL translation cannot express at
// all: an allowed-value set carrying its own numbering.
func TestNumberedEnum(t *testing.T) {
	dir := write(t, map[string]string{"root.vspec": `
Vehicle:
  type: branch
  description: Root.
  RoadSurface:
    datatype: uint8
    type: sensor
    description: Surface condition.
    enum:
      UNKNOWN: 0
      DRY: 1
      WET: 2
`})
	root, err := Parse(filepath.Join(dir, "root.vspec"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	surface, _ := root.Child("RoadSurface")
	want := []EnumEntry{{"UNKNOWN", 0}, {"DRY", 1}, {"WET", 2}}
	if len(surface.Enum) != len(want) {
		t.Fatalf("enum = %v", surface.Enum)
	}
	for i, e := range want {
		if surface.Enum[i] != e {
			t.Errorf("enum[%d] = %v, want %v", i, surface.Enum[i], e)
		}
	}
}

// TestRejectsUnknownShape is the rule the package exists to enforce: an
// unfamiliar construct is an error, never a silently skipped node.
func TestRejectsUnknownShape(t *testing.T) {
	dir := write(t, map[string]string{"root.vspec": `
Vehicle:
  description: A node with no type.
`})
	if _, err := Parse(filepath.Join(dir, "root.vspec")); err == nil {
		t.Fatal("Parse accepted a node with no type; want an error")
	}
}

// TestIncludeIntoMissingNode checks a typo in an attachment point fails
// rather than inventing a branch nothing expects.
func TestIncludeIntoMissingNode(t *testing.T) {
	dir := write(t, map[string]string{
		"root.vspec":  "Vehicle:\n  type: branch\n  description: Root.\n#include Other.vspec Vehicle.Nope\n",
		"Other.vspec": "Thing:\n  type: branch\n  description: A thing.\n",
	})
	if _, err := Parse(filepath.Join(dir, "root.vspec")); err == nil {
		t.Fatal("Parse accepted an include into a node that does not exist")
	}
}
