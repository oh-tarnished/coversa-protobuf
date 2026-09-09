// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package model

// enums.go names the emitted enums.
//
// # Why the name is shortened
//
// An allowed-value set belongs to the signal that constrains it, and VSS
// identifies that signal by its whole path:
// `Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.SupportedInletTypes`.
// Carried through, that produced a 64-character type and — because AIP-126
// prefixes every value with its enum's name — 91-character constants.
//
// Nearly all of that prefix is already stated by the package the enum lands
// in, so the schema was saying the same thing twice. The name here is the
// shortest trailing run of path segments unique *within its package*, which
// is the smallest name a reader can resolve without leaving the file.
//
// # The cost, stated plainly
//
// A shortest-unique name is not stable against additions: a future VSS
// release adding a sibling signal called `Level` would force the existing
// `Level` to lengthen, and that is a breaking rename caused by an unrelated
// addition. `just survey` reports it and `buf breaking` fails on it, so it
// cannot ship silently — but it is a real cost, accepted for names a person
// can read.

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// nameEnums resolves every enum's emitted name, package by package.
func (m *Model) nameEnums() {
	m.enumNames = map[string]string{}
	for _, pkg := range m.Packages {
		for fqn, name := range resolveEnumNames(pkg) {
			m.enumNames[fqn] = name
		}
	}
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
		name := naming.Pascal(t.Name)
		taken[name] = true

		// `<Message>State` is unavailable too, even though nothing declares
		// it. AIP-216 reads a package-level `ConvertibleState` beside a
		// message `Convertible` as a lifecycle enum that should have been
		// nested, and rejects it. Nesting is the linter's remedy and is not
		// taken here — a nested enum is a different type path in every target
		// IDL this schema generates — so the name is skipped.
		taken[name+"State"] = true
	}

	enums := pkg.Enums()
	for depth := 1; depth <= maxDepth(enums); depth++ {
		out, used := map[string]string{}, map[string]bool{}

		ok := true
		for _, e := range enums {
			name := enumNameAt(e, depth)
			if used[name] || taken[name] {
				ok = false
				break
			}
			used[name] = true
			out[e.FQN] = name
		}
		if ok {
			return out
		}
	}

	// Every depth collided: two signals share their whole path, which cannot
	// happen. Fall back to the full path rather than emitting a duplicate.
	out := map[string]string{}
	for _, e := range enums {
		out[e.FQN] = enumNameAt(e, len(strings.Split(e.FQN, ".")))
	}
	return out
}

// maxDepth is the longest path any enum in the set has, and so the deepest
// the search needs to go.
func maxDepth(enums []*vspec.Node) int {
	longest := 1
	for _, e := range enums {
		if n := len(strings.Split(e.FQN, ".")); n > longest {
			longest = n
		}
	}
	return longest
}

// enumNameAt renders an enum using the last depth segments of its path.
func enumNameAt(n *vspec.Node, depth int) string {
	// A source that named its own enum keeps that name: GraphQL declares
	// `AgeRange` as a type, and shortening a name someone chose would be
	// renaming rather than de-duplicating.
	if n.EnumName != "" {
		return stateSuffix(naming.Pascal(n.EnumName))
	}

	parts := strings.Split(n.FQN, ".")
	if depth < len(parts) {
		parts = parts[len(parts)-depth:]
	}
	return stateSuffix(naming.Pascal(strings.Join(parts, "_")))
}

// stateSuffix applies AIP-216, which prefers "State" over "Status" and reads
// any enum ending in Status as a lifecycle enum.
//
// VSS uses Status for a physical position — a roof, a massage programme, a
// seat's occupancy — so the suffix is renamed and the field naming the enum
// keeps VSS's own term.
func stateSuffix(name string) string {
	if strings.HasSuffix(name, "Status") {
		return strings.TrimSuffix(name, "Status") + "State"
	}
	return name
}
