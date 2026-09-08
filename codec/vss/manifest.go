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
	// Version is the specification revision the schema was generated from,
	// YYYY-MM-DD. A manifest and a message must agree on it: a field added in
	// a later revision is absent from an earlier manifest, and converting
	// with the wrong one silently drops it.
	Version string `json:"version"`

	// Commit is the upstream revision the schema was generated from.
	Commit string `json:"commit"`

	// Resources maps each protobuf message's full name to its mapping.
	Resources map[string]*Resource `json:"resources"`
}

// Resource is one protobuf message and where it came from.
type Resource struct {
	// FQN is the fully qualified VSS name of the branch, e.g.
	// "Vehicle.Cabin.Seat". Empty for a message VSS does not declare.
	FQN string `json:"fqn,omitempty"`

	// Pattern is the AIP resource name pattern, where the message is a
	// resource: "vehicles/{vehicle}/seats/{seat}".
	Pattern string `json:"pattern,omitempty"`

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
	// This is the half of the mapping a consumer cannot derive: `production`
	// is `productionDate` upstream, and nothing in the protobuf says so.
	Source string `json:"source,omitempty"`

	// Element is the VSS kind: "attribute", "sensor" or "actuator". Only an
	// actuator is writable, which a converter feeding data *into* a vehicle
	// needs to respect.
	//
	// Two values are this schema's own rather than VSS's. "branch" marks a
	// field holding a nested message. "instance_axis" marks one of the axes
	// that say *which* member of a repeated branch a reading belongs to --
	// VSS expresses that by expanding the path rather than by a signal, so
	// there is no upstream kind to report.
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

	// Values maps each emitted enum constant to its source spelling, for a
	// field whose type is an enum.
	//
	// The two differ more often than not: VSS writes "Row1" where the schema
	// emits SEAT_INSTANCE_TAG_DIMENSION1_ROW1, and a VSS-native peer sends
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
