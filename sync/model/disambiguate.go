// Copyright 2026 Srikanth Kandarpa.
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
// The message follows the qualification, because AIP-123 ties it to the type:
// a resource's `singular` must be the lower camel case of its message name, so
// a `chassisAxles` collection of `Axle` is a violation the linter reports three
// times over. Provenance is not lost -- the branch annotation still carries
// `Vehicle.Chassis.Axle`, which is what a consumer joins on.
//
// The collection segment is left alone wherever the parent already separates
// it. A wheel hangs beneath an axle that is now named, so
// `chassisAxles/{chassis_axle}/wheels/{wheel}` identifies exactly one
// resource and `chassisAxleWheels` would only restate the segment before it.
// That is AIP-122's nested collections, and AIP-123 exempts it from the rule
// that a collection segment matches the plural.
//
// The *type* takes the qualification anyway, because the two namespaces are
// not the same shape. A collection segment is scoped by the segment before
// it; a resource type is flat across the API, and AIP-123 requires it unique
// within one. Leaving the three wheels as `vdm.covesa.org/Wheel` made every
// `resource_reference` naming that type ambiguous between three resources
// that share no field -- a brake wheel carries torque limits, a chassis wheel
// a tire, a suspension wheel a damping rate -- and named all three services
// `Wheels`. So `ChassisAxleWheel` is addressed at `.../wheels/{wheel}`,
// exactly as `merchantapi.googleapis.com/AccountIssue` is addressed at
// `accounts/{account}/issues/{issue}`.

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
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

			// The path axis: qualified only where a sibling sharing this leaf
			// hangs beneath the same parent, since anywhere else the segment
			// before it already separates the two.
			segment := name
			if len(group) > 1 && p.qualified != "" {
				segment = p.qualified
			}

			// The type axis: qualified wherever the package was, because a
			// type name is unique across the API or it is ambiguous.
			//
			// The node's Name is what the message is rendered from, and
			// AIP-123 requires it to match the singular. FQN is the node's
			// identity and is untouched, so every lookup, rename and
			// annotation still resolves against the path VSS published.
			if p.qualified != "" {
				name = p.qualified
				p.Root.Name = naming.Pascal(name)
			}
			p.setNames(name, segment)
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

// setNames gives a package the two name axes: the AIP singular and plural,
// which follow the resource type, and the collection and identifier segments,
// which follow the path. They differ only for a nested collection.
//
// Snake case in on both, since the leaf and the qualified name are.
func (p *Package) setNames(typeName, segment string) {
	p.Singular = naming.LowerCamel(typeName)
	p.Plural = naming.Pluralise(p.Singular)
	p.ID = naming.LowerCamel(segment)
	p.Segment = naming.Pluralise(p.ID)
}
