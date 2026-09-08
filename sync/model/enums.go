// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// enums.go names the emitted enums.
//
// # Why the name is shortened
//
// The source model names an allowed-value enum after the whole VSS path:
// `Vehicle_Powertrain_TractionBattery_Charging_ChargingPort_SupportedInletTypes_Enum`.
// Carried through, that produced a 64-character type and, because AIP-126
// prefixes every value with its enum's name, 91-character constants.
//
// Nearly all of that prefix is already stated by the package the enum lands
// in -- `protobuf.covesa.vss.propulsion.charging_port.v1` -- so the schema was
// saying the same thing twice. The name here is the shortest trailing run of
// path segments that is unique *within its package*, which is the smallest
// name a reader can resolve without leaving the file.
//
// # The cost, stated plainly
//
// A shortest-unique name is not stable against additions. A future VSS
// release adding a sibling signal called `Level` would force the existing
// `Level` to lengthen, and that is a breaking rename caused by an unrelated
// addition. `just survey` reports it and `buf breaking` fails on it, so it
// cannot ship silently -- but it is a real cost, accepted for names a person
// can read.

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/naming"
)

// enumSuffix is what the source model appends to an allowed-value enum.
const enumSuffix = "_Enum"

// nameEnums resolves every enum's emitted name, package by package.
func (m *Model) nameEnums() {
	m.enumNames = map[string]string{}
	for _, pkg := range m.Packages {
		for source, name := range resolveEnumNames(pkg) {
			m.enumNames[source] = name
		}
	}
}

// EnumName is the protobuf name emitted for a source enum.
//
// Falls back to the full sanitised name for an enum no package claimed, which
// should not happen -- but a missing entry must not produce an empty type
// name.
func (m *Model) EnumName(source string) string {
	if n, ok := m.enumNames[source]; ok {
		return n
	}
	return enumBase(source)
}

// resolveEnumNames picks the shortest unique name for each enum in a package.
//
// Uniqueness is checked against the package's message names as well as its
// other enums: proto3 scopes every declaration in the package, so an enum
// named `Level` collides with a message named `Level` exactly as it would
// with another enum.
func resolveEnumNames(pkg *Package) map[string]string {
	taken := map[string]bool{}
	for _, t := range pkg.Types {
		name := MessageName(t.Name)
		taken[name] = true

		// `<Message>State` is unavailable too, even though nothing declares
		// it. AIP-216 reads a package-level `ConvertibleState` beside a
		// message `Convertible` as a lifecycle enum that should have been
		// nested as `Convertible.State`, and rejects it. Nesting is the
		// linter's remedy and is not taken here -- a nested enum is a
		// different type path in every target IDL this schema generates --
		// so the name is skipped and the search goes one segment deeper.
		taken[name+"State"] = true
	}

	out := make(map[string]string, len(pkg.Enums))
	for depth := 1; depth <= maxSegments(pkg); depth++ {
		out = map[string]string{}
		used := map[string]bool{}

		ok := true
		for _, e := range pkg.Enums {
			name := enumNameAt(e.Name, depth)
			if used[name] || taken[name] {
				ok = false
				break
			}
			used[name] = true
			out[e.Name] = name
		}
		if ok {
			return out
		}
	}

	// Every depth collided, which means two enums share their whole path.
	// Fall back to the full name rather than emitting a duplicate.
	for _, e := range pkg.Enums {
		out[e.Name] = enumBase(e.Name)
	}
	return out
}

// maxSegments is the longest path any enum in the package has, and therefore
// the deepest the search needs to go.
func maxSegments(pkg *Package) int {
	longest := 1
	for _, e := range pkg.Enums {
		if n := len(segments(e.Name)); n > longest {
			longest = n
		}
	}
	return longest
}

// enumNameAt renders an enum using its last `depth` path segments.
func enumNameAt(source string, depth int) string {
	parts := segments(source)
	if depth < len(parts) {
		parts = parts[len(parts)-depth:]
	}
	return stateSuffix(naming.Pascal(strings.Join(parts, "_")))
}

// enumBase is the enum's full name, with the source model's decoration
// removed: every allowed-value enum is prefixed `Vehicle_` and suffixed
// `_Enum`, and neither says anything the package does not.
func enumBase(source string) string {
	return stateSuffix(naming.Pascal(strings.Join(segments(source), "_")))
}

// segments splits a source enum name into its VSS path components.
func segments(source string) []string {
	s := strings.TrimPrefix(source, "Vehicle_")
	s = strings.TrimSuffix(s, enumSuffix)
	return strings.Split(s, "_")
}

// stateSuffix applies AIP-216, which prefers "State" over "Status" and reads
// any enum ending in Status as a lifecycle enum.
//
// VSS uses Status for a physical position -- a roof, a massage programme, a
// seat's occupancy -- so the suffix is renamed and the field naming the enum
// keeps VSS's own term through the catalogue.
func stateSuffix(name string) string {
	if strings.HasSuffix(name, "Status") {
		return strings.TrimSuffix(name, "Status") + "State"
	}
	return name
}
