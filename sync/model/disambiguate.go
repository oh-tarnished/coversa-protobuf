// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// disambiguate.go resolves branches that share a name at different paths.
//
// VSS names a branch for what it is, not for where it sits, so `Axle` occurs
// three times: under Chassis, under MotionManagement.Brake and under
// MotionManagement.Suspension. They are three different branches carrying
// different signals -- a chassis axle has a wheel count and a track width, a
// brake axle has torque limits -- and each is a resource of its own.
//
// The leaf alone cannot name all three. Both things derived from it collide:
// the package directory, so three packages write `axle.proto` over each other
// and only the last survives; and the resource name, so
// `vehicles/{v}/axles/{a}` identifies three different resources at once,
// which AIP-123 forbids.
//
// The fix is the one package enums already use: the shortest trailing run of
// path segments that is unique. `Vehicle.Chassis.Axle` becomes `chassis_axle`
// and `Vehicle.MotionManagement.Brake.Axle` becomes `brake_axle`, while every
// branch with an unambiguous name keeps it.
//
// The *message* keeps VSS's own term either way. `Axle` is the name in three
// packages, which proto3 allows and which keeps the mapping into the other
// target IDLs on the specification's vocabulary; only the directory is
// qualified, because only the directory has to be unique.
//
// The collection segment is left alone wherever the parent already separates
// it. A wheel hangs beneath an axle that is now named, so
// `chassisAxles/{chassis_axle}/wheels/{wheel}` identifies exactly one
// resource and `chassisAxleWheels` would only restate the segment before it.

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/naming"
)

// disambiguate renames every package whose leaf name another package shares.
//
// Runs after the whole tree is partitioned, because a name is only ambiguous
// relative to the others: nothing about `Vehicle.Chassis.Axle` on its own
// says it needs qualifying.
func (m *Model) disambiguate() {
	groups := map[string][]*Package{}
	for _, p := range m.Packages {
		groups[p.Name] = append(groups[p.Name], p)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, p := range group {
			p.qualified = uniqueName(p, group)
			p.Name, p.Dir = p.qualified, p.qualified
		}
	}
	m.rebuildPatterns()
}

// rebuildPatterns gives every package its resource name, qualifying the
// collection segment only where the leaf would be ambiguous.
//
// Ambiguity here is structural rather than textual: a collection is ambiguous
// when a package sharing its leaf name hangs beneath the same parent. Three
// axles all hang off the vehicle, so all three are qualified -- leaving one
// of them the bare `axles` would say it is *the* axle collection, and which
// one got it would come down to the order the tree was walked in. The three
// wheels hang off three different axles, so `wheels` beneath each names
// exactly one thing and qualifying it would only restate the segment before.
func (m *Model) rebuildPatterns() {
	// The naming parent is a fact about the tree, not about any name, so it
	// is settled first and the question below can be asked without a
	// half-built pattern to read.
	for _, p := range m.Packages {
		p.NameParent = p.nameParent()
	}

	siblings := map[[2]any][]*Package{}
	for _, p := range m.Packages {
		key := [2]any{p.NameParent, leaf(p.Root.FQN)}
		siblings[key] = append(siblings[key], p)
	}
	for _, group := range siblings {
		for _, p := range group {
			name := leaf(p.Root.FQN)
			if len(group) > 1 && p.qualified != "" {
				name = p.qualified
			}
			p.setNames(name)
		}
	}

	// Top down: a pattern quotes its parent's, and spawn appends a package
	// before its own children, so one pass reaches every child after its
	// parent is final.
	for _, p := range m.Packages {
		p.setPattern()
	}
}

// uniqueName is the shortest trailing run of p's path that no other package in
// the group shares.
func uniqueName(p *Package, group []*Package) string {
	parts := strings.Split(p.Root.FQN, ".")

	for depth := 2; depth <= len(parts); depth++ {
		name := nameAt(parts, depth)

		unique := true
		for _, other := range group {
			if other == p {
				continue
			}
			if nameAt(strings.Split(other.Root.FQN, "."), depth) == name {
				unique = false
				break
			}
		}
		if unique {
			return name
		}
	}

	// Two branches share their whole path, which the tree cannot produce: a
	// fully qualified name is unique by construction.
	return nameAt(parts, len(parts))
}

// nameAt renders the last depth segments of a path as a snake_case name.
func nameAt(parts []string, depth int) string {
	if depth < len(parts) {
		parts = parts[len(parts)-depth:]
	}
	return naming.Snake(strings.Join(parts, "_"))
}

// setNames gives a package the singular and plural its resource name is built
// from. Snake case in, since both the leaf and the qualified name are.
func (p *Package) setNames(name string) {
	p.Singular = naming.LowerCamel(name)
	p.Plural = naming.Pluralise(p.Singular)
}
