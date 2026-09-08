// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// partition.go decides which types become packages of their own and which
// stay embedded, then walks the tree assigning every type to exactly one
// place.
//
// The rule, from CLAUDE.md rule 7: a branch directly beneath a root is a
// package; deeper down, only a branch whose members have identity.

import (
	"github.com/the-protobuf-project/vdm/sync/internal/naming"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// spawn promotes the child resources of one type into packages of their own,
// then recurses into each so a branch nested three deep is still addressable.
//
// The depth is the source model's, not a choice made here: a wheel belongs to
// an axle and an axle to a chassis, so
// `vehicles/{v}/chassisAxles/{a}/axleWheels/{w}` is the name the domain
// already has.
func (m *Model) spawn(root *sdl.Def, parent *Package, fam Family, claimed map[string]bool) {
	for _, f := range root.Fields {
		bt, ok := m.Types[f.Type.Name]
		if !ok || claimed[bt.Name] {
			continue
		}

		// A repeated branch, or one the source model gives an instance tag,
		// has members with identity -- which is exactly what that tag exists
		// to declare.
		identifiable := f.Type.List || m.hasInstanceTag(bt)

		if !m.promotes(parent, bt, identifiable) {
			// Not a resource itself, but it may contain one: AmbientLight
			// hangs beneath Cabin.Light, and Light is a grouping node with a
			// single occurrence. Walk through it, attaching anything
			// identifiable to the nearest promoted ancestor.
			//
			// The intermediate name is dropped from the resource name --
			// "vehicles/{v}/ambientLights/{x}", not ".../light/..." --
			// because a segment with one possible value identifies nothing.
			// The full VSS path survives in the branch annotation's `fqn`.
			m.spawn(bt, parent, fam, claimed)
			continue
		}

		pkg := newPackage(bt, parent, fam, f.Type.List)
		pkg.setPattern(identifiable)
		m.Packages = append(m.Packages, pkg)

		claimed[bt.Name] = true
		pkg.Types = append(pkg.Types, bt)
		// Promote this branch's own children before collecting, so a nested
		// resource is claimed by its own package rather than absorbed here.
		m.spawn(bt, pkg, fam, claimed)
		m.collect(bt, pkg, claimed)
	}
}

// promotes reports whether a referenced type becomes a package of its own.
//
// Directly beneath a root, every branch is promoted: the cabin, the
// powertrain and the chassis are the top-level areas a consumer addresses,
// whether or not a vehicle has more than one. Deeper down, only a branch
// whose members have identity is -- a singular branch below the top level is
// a grouping node with one occurrence, so it stays inside its parent's
// package where its fields can be read in the same call.
func (m *Model) promotes(parent *Package, t *sdl.Def, identifiable bool) bool {
	if parent.Parent == nil {
		return isBranch(t) || identifiable
	}
	return identifiable
}

// newPackage creates the package for a promoted branch, inheriting its
// ancestor's domain or, at the top level, taking one from the table.
func newPackage(t *sdl.Def, parent *Package, fam Family, isList bool) *Package {
	domain := parent.Domain
	if domain == "" && fam == FamilyVSS {
		domain = domainOf(t.Name)
	}
	singular := naming.LowerCamel(t.Name)
	return &Package{
		Family: fam, Domain: domain,
		Name: naming.Snake(t.Name), Dir: naming.Snake(t.Name), Root: t,
		Singular: singular, Plural: naming.Pluralise(singular),
		IsList: isList, Parent: parent,
	}
}

// setPattern gives the package its resource name.
//
// AIP-123 requires a name to alternate collection and identifier. A
// singleton's name ends in a literal, so a child hung beneath one would put
// two literals in a row and break the alternation. An identifiable resource
// therefore hangs off the nearest ancestor ending in an identifier -- for a
// seat, the vehicle rather than the cabin.
//
// Nothing is lost: the cabin has exactly one occurrence, so a "cabin" segment
// would identify nothing, and the full VSS path is in the branch annotation.
func (p *Package) setPattern(identifiable bool) {
	if p.IsList || identifiable {
		np := p.namingParent()
		p.Pattern = np.Pattern + "/" + p.Plural + "/{" + naming.Snake(p.Root.Name) + "}"
		p.NameParent = np
		return
	}
	// The parent holds exactly one, so it is a singleton, AIP-156: its name
	// has no id segment.
	p.Pattern = p.Parent.Pattern + "/" + p.Singular
	p.NameParent = p.Parent
}

// namingParent is the nearest ancestor whose name ends in an identifier.
func (p *Package) namingParent() *Package {
	a := p.Parent
	for a != nil && a.Parent != nil && !a.IsList {
		a = a.Parent
	}
	return a
}

// hasInstanceTag reports whether the source model declares a branch's members
// individually identifiable.
func (m *Model) hasInstanceTag(d *sdl.Def) bool {
	if _, ok := m.Types[d.Name+"_InstanceTag"]; ok {
		return true
	}
	for _, f := range d.Fields {
		if f.Name == "instanceTag" {
			return true
		}
	}
	return false
}

// isBranch reports whether a type is marked a VSS branch or struct.
func isBranch(d *sdl.Def) bool {
	v, ok := d.Directive("vspec")
	if !ok {
		return false
	}
	e, _ := v.Arg("element")
	return e == "BRANCH" || e == "STRUCT"
}
