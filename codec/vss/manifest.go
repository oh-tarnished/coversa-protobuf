// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package vss converts between this schema and the COVESA Vehicle Data
// Model's own shape.
//
// # What the manifest is for
//
// The schema records where every field came from -- its VSS fully qualified
// name, its unit, the source spelling of every enum value -- in the
// annotations under protobuf.covesa.vss.annotations.v1. Those annotations are
// enough to reconstruct the original shape, but reading them means walking a
// protobuf descriptor, which is a different job in every language.
//
// The manifest is that mapping, extracted once into plain JSON. A consumer in
// any language reads it and can convert a message to the VSS tree a
// VSS-native system expects, without a protobuf toolchain and without
// reimplementing this repository's naming rules.
//
// The Go functions in this package are the reference implementation of that
// conversion, and exist partly to prove the manifest is sufficient: a mapping
// nothing round-trips against is a mapping with untested gaps.
package vss

// Manifest is the whole mapping, as written to codec/manifest.json.
type Manifest struct {
	// Spec is the revisions the schema was generated from. A manifest and a
	// message must agree on them: a field added in a later revision is absent
	// from an earlier manifest, and converting with the wrong one silently
	// drops it.
	Spec Spec `json:"spec"`

	// Resources maps each protobuf message's full name to its mapping.
	Resources map[string]*Resource `json:"resources"`
}

// Spec is the two upstream revisions, each pinned separately because each
// moves separately: the vehicle tree comes from the Vehicle Signal
// Specification, everything around it from the Vehicle Data Model.
type Spec struct {
	VSS Revision `json:"vss"`
	VDM Revision `json:"vdm"`
}

// Revision is one pinned upstream specification.
type Revision struct {
	// Version is the revision date, YYYY-MM-DD.
	Version string `json:"version"`

	// Commit is the upstream revision actually read.
	Commit string `json:"commit"`
}

// String renders the pair the way an error message names them.
func (s Spec) String() string {
	return "vss " + s.VSS.Version + ", vdm " + s.VDM.Version
}

// Resource is one protobuf message and where it came from.
type Resource struct {
	// FQN is the fully qualified VSS name of the branch, e.g.
	// "Vehicle.Cabin.Seat". Empty for a message VSS does not declare.
	FQN string `json:"fqn,omitempty"`

	// Pattern is the AIP resource name pattern, where the message is a
	// resource: "vehicles/{vehicle}/seats/{seat}".
	Pattern string `json:"pattern,omitempty"`

	// Instances are the axes the branch expands across, where VSS gives it
	// any: [["Row1","Row2"],["DriverSide","Middle","PassengerSide"]].
	//
	// This is how a resource id maps back to a VSS path. A seat named
	// `row1DriverSide` is `Vehicle.Cabin.Seat.Row1.DriverSide` upstream, and
	// nothing in the protobuf says how to take the id apart -- the axes are
	// the only record of which segments it was built from, and in what order.
	Instances [][]string `json:"instances,omitempty"`

	// Fields maps each protobuf field name to its mapping.
	Fields map[string]*Field `json:"fields"`
}

// Field is one protobuf field and where it came from.
type Field struct {
	// FQN is the signal's fully qualified VSS name, e.g.
	// "Vehicle.Cabin.Seat.Height".
	FQN string `json:"fqn,omitempty"`

	// Source is the field's name in the source model, where the schema
	// renamed it to satisfy an AIP rule. Absent when the two agree.
	//
	// The VSS path segment, PascalCase, as it appears in FQN. This is the
	// half of the mapping a consumer cannot derive: `production` is
	// `ProductionDate` upstream, and nothing in the protobuf says so.
	Source string `json:"source,omitempty"`

	// Element is the VSS kind: "attribute", "sensor" or "actuator". Only an
	// actuator is writable, which a converter feeding data *into* a vehicle
	// needs to respect.
	//
	// One value is this schema's own rather than VSS's: "branch" marks a
	// field holding a nested message, which VSS models as a node rather than
	// as a signal with a kind.
	Element string `json:"element,omitempty"`

	// Unit is the symbol the value is expressed in, e.g. "km/h". Absent for a
	// field that carries no unit.
	//
	// The schema pins one unit per signal, so this is a statement about what
	// the number means rather than a choice the message carries.
	Unit string `json:"unit,omitempty"`

	// Quantity is the physical quantity Unit measures, e.g. "velocity".
	Quantity string `json:"quantity,omitempty"`

	// Enum names the enum type, where the field is one.
	Enum string `json:"enum,omitempty"`

	// Message is the full protobuf name of the nested message, where the
	// field holds one.
	//
	// Without it a converter stops at the boundary and hands back the
	// protobuf spelling of everything inside -- a nested enum would arrive as
	// AIR_DISTRIBUTION_SIDE_WINDOW where a VSS-native peer expects
	// "SIDE_WINDOW".
	Message string `json:"message,omitempty"`

	// Values maps each emitted enum constant to its source spelling, for a
	// field whose type is an enum.
	//
	// The two differ more often than not: VSS writes "SIDE_WINDOW" where the
	// schema emits AIR_DISTRIBUTION_SIDE_WINDOW, and a VSS-native peer sends
	// the former.
	Values map[string]string `json:"values,omitempty"`
}

// Lookup returns the mapping for one field of one message.
func (m *Manifest) Lookup(message, field string) (*Field, bool) {
	r, ok := m.Resources[message]
	if !ok {
		return nil, false
	}
	f, ok := r.Fields[field]
	return f, ok
}
