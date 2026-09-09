// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package vspec parses the COVESA Vehicle Signal Specification.
//
// # What a .vspec file is
//
// YAML, plus a preprocessor VSS layers on top of it. Each top-level key is a
// node; a node is either a `branch` (a grouping) or a signal — `sensor`,
// `actuator` or `attribute`. Nodes nest by YAML nesting, and across files by
// an `#include` directive that grafts one file's tree onto a named point of
// another's.
//
// The YAML itself is decoded by a library. The rest -- include resolution and
// instance expansion -- is VSS's own and is implemented here.
//
// # Why this rather than the GraphQL translation
//
// COVESA also publishes the tree as GraphQL SDL, inside the Vehicle Data
// Model, and this generator read that first. It is lossy: 169 `comment`
// fields, most `default` values and every `pattern` constraint are dropped in
// translation, and names are damaged by case-splitting -- "UNIX Timestamp"
// arrives as `U_N_I_X_TIMESTAMP`, "GNSS" as `GN_SS`. Reading the
// specification directly recovers all of it.
package vspec

// Kind is what a node is: a grouping, or one of the three signal kinds.
type Kind string

const (
	// KindBranch is a grouping node with children.
	KindBranch Kind = "branch"

	// KindSensor is a value the vehicle measures and reports.
	KindSensor Kind = "sensor"

	// KindActuator is a value the vehicle accepts and acts on. The only
	// writable kind.
	KindActuator Kind = "actuator"

	// KindAttribute is a value fixed for the life of the vehicle.
	KindAttribute Kind = "attribute"

	// KindStruct is a record read and written as one value, and KindProperty
	// one of its members. VSS uses both sparingly.
	KindStruct   Kind = "struct"
	KindProperty Kind = "property"
)

// Signal reports whether the kind carries a value rather than grouping others.
func (k Kind) Signal() bool {
	switch k {
	case KindSensor, KindActuator, KindAttribute, KindProperty:
		return true
	}
	return false
}

// EnumEntry is one named value of an explicitly numbered enum.
type EnumEntry struct {
	Name   string
	Number int
}

// Node is one entry in the specification tree.
type Node struct {
	// Name is the node's own name, e.g. "IsBelted".
	Name string

	// FQN is the fully qualified path, e.g. "Vehicle.Cabin.Seat.IsBelted".
	// This is the identity VSS itself uses.
	FQN string

	// Kind is branch, sensor, actuator, attribute, struct or property.
	Kind Kind

	// Description is the one-line summary every node carries.
	Description string

	// Comment is the longer note VSS attaches where behaviour needs
	// explaining. The GraphQL translation drops all 169 of these.
	Comment string

	// Datatype is the VSS type name for a signal: "float", "uint8",
	// "string[]" and so on. Empty on a branch.
	Datatype string

	// Unit is the unit the value is expressed in, e.g. "mm". Empty where the
	// quantity is dimensionless.
	Unit string

	// Min and Max are the declared bounds, as written. Kept as strings
	// because a bound may be an integer or a float and the datatype decides
	// which.
	Min string
	Max string

	// Allowed is the set of permitted values, which becomes an enum.
	Allowed []string

	// Enum is an allowed-value set that carries its own numbering:
	//
	//	enum:
	//	  UNKNOWN: 0
	//	  DRY: 1
	//
	// Richer than Allowed, and the numbers are the specification's own, so a
	// protobuf enum can adopt them rather than inventing a sequence. The
	// GraphQL translation has no way to express this.
	Enum []EnumEntry

	// Default is the value assumed when none is reported.
	Default string

	// Pattern is a regular expression the value must match.
	Pattern string

	// Deprecation is the note VSS attaches to a signal on its way out.
	Deprecation string

	// Repeated marks a node the parent holds many of.
	//
	// VSS says this with `instances`, which also names the occurrences. A
	// converted GraphQL list has no names, so it sets this instead.
	Repeated bool

	// Required marks a node the source declares non-null. VSS has no such
	// concept; only converted GraphQL sets it.
	Required bool

	// Ref names a resource this node points at rather than contains.
	//
	// VSS has no such node: its tree is pure containment. The Vehicle Data
	// Model does -- a charging session names the vehicle that charged -- and
	// that association becomes a resource name rather than an embedded copy,
	// because AIP-215 forbids the copy and because a session is a historical
	// fact while the vehicle it names keeps changing.
	Ref string

	// EnumName is the name the source gave an allowed-value set, where it
	// gave one. VSS names enums after the signal that owns them, so this is
	// empty for everything read from .vspec.
	EnumName string

	// Instances are the axes this branch expands across, already expanded
	// from VSS's `Row[1,2]` shorthand into explicit names.
	Instances [][]string

	// Children are the nodes beneath this one, in specification order.
	Children []*Node

	// File is where the node was declared, for error messages.
	File string
}

// Walk calls fn for this node and every node beneath it, depth first.
func (n *Node) Walk(fn func(*Node)) {
	fn(n)
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// Child returns the named child, if present.
func (n *Node) Child(name string) (*Node, bool) {
	for _, c := range n.Children {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}
