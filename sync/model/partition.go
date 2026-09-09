// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// partition.go decides which branches become packages of their own and which
// stay embedded, then walks the tree assigning each to exactly one place.
//
// The rule, from CLAUDE.md rule 7: a branch directly beneath a root is a
// package; deeper down, only a branch whose members have identity.

import (
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// spawn promotes a branch's child resources into packages of their own, then
// recurses so a branch nested three deep is still addressable.
//
// The depth is the specification's, not a choice made here: a wheel belongs
// to an axle and an axle to a chassis, so
// `vehicles/{v}/chassisAxles/{a}/axleWheels/{w}` is the name VSS already has.
func (m *Model) spawn(parent *vspec.Node, pkg *Package, fam Family, claimed map[string]bool) {
	for _, child := range parent.Children {
		if child.Kind != vspec.KindBranch || claimed[child.FQN] || child.Ref != "" {
			continue
		}

		// A branch VSS gives instances to has members with identity — that is
		// what `instances` exists to declare. A repeated branch says the same
		// thing without naming the occurrences.
		identifiable := len(child.Instances) > 0 || child.Repeated

		if !m.promotes(pkg, identifiable) {
			// Not a resource itself, but it may contain one: an ambient light
			// hangs beneath Cabin.Light, and Light has a single occurrence.
			// Walk through it, attaching anything identifiable to the nearest
			// promoted ancestor.
			//
			// The intermediate name drops out of the resource name —
			// `vehicles/{v}/ambientLights/{x}`, not `.../light/...` — because
			// a segment with one possible value identifies nothing. The full
			// VSS path survives in the branch annotation.
			m.spawn(child, pkg, fam, claimed)
			continue
		}

		sub := newPackage(child, pkg, fam, identifiable)
		m.Packages = append(m.Packages, sub)
		claimed[child.FQN] = true
		sub.Types = append(sub.Types, child)

		// Promote this branch's own children before collecting, so a nested
		// resource is claimed by its own package rather than absorbed here.
		m.spawn(child, sub, fam, claimed)
		m.collect(child, sub, claimed)
	}
}

// promotes reports whether a branch becomes a package of its own.
//
// Directly beneath a root every branch is promoted: the cabin, the powertrain
// and the chassis are the top-level areas a consumer addresses, whether or
// not a vehicle has more than one. Deeper down only an identifiable branch
// is — a singular branch below the top level is a grouping node with one
// occurrence, so it stays where its fields can be read in the same call.
func (m *Model) promotes(parent *Package, identifiable bool) bool {
	if parent.Parent == nil {
		return true
	}
	return identifiable
}

// newPackage creates the package for a promoted branch, inheriting its
// ancestor's domain or, at the top level, taking one from the table.
func newPackage(n *vspec.Node, parent *Package, fam Family, identifiable bool) *Package {
	domain := parent.Domain
	if domain == "" && fam == FamilyVSS {
		domain = domainOf(n.Name)
	}

	singular := naming.LowerCamel(n.Name)
	pkg := &Package{
		Family: fam, Domain: domain,
		Name: naming.Snake(n.Name), Dir: naming.Snake(n.Name),
		Root: n, Singular: singular, Plural: naming.Pluralise(singular),
		IsList: identifiable, Parent: parent,
	}
	pkg.setPattern()
	return pkg
}

// setPattern gives the package its resource name.
//
// AIP-123 requires a name to alternate collection and identifier. A
// singleton's name ends in a literal, so a child hung beneath one would put
// two literals in a row. An identifiable resource therefore hangs off the
// nearest ancestor ending in an identifier — for a seat, the vehicle rather
// than the cabin.
//
// Nothing is lost: the cabin has one occurrence, so a `cabin` segment would
// identify nothing, and the full VSS path is in the branch annotation.
func (p *Package) setPattern() {
	switch {
	case p.NameParent == nil:
		p.Pattern = p.Plural + "/{" + p.Singular + "}"
	case p.IsList:
		p.Pattern = p.NameParent.Pattern + "/" + p.Plural + "/{" + naming.Snake(p.Singular) + "}"
	default:
		// The parent holds exactly one, so it is a singleton, AIP-156: its
		// name has no id segment.
		p.Pattern = p.NameParent.Pattern + "/" + p.Singular
	}
}

// nameParent is the resource this one's name hangs beneath: for a collection,
// the nearest ancestor whose own name ends in an identifier; for a singleton,
// its parent; for a root, nothing.
//
// Structural -- it reads IsList and the parent chain, never a name -- so it
// can be settled before any name is.
func (p *Package) nameParent() *Package {
	if !p.IsList {
		return p.Parent
	}
	a := p.Parent
	for a != nil && a.Parent != nil && !a.IsList {
		a = a.Parent
	}
	return a
}
