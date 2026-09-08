// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// partition.go decides which types become packages of their own and which stay
// embedded, and walks the tree assigning every type to exactly one place.
//
// The rule it implements is in CLAUDE.md rule 7: a branch directly beneath a
// root is a package; deeper down, only a branch whose members have identity.

// spawnChildren promotes each of a root's child resources into its own
// package.
//
// A referenced object type is a child *resource* rather than an embedded value
// object when the source model marks it a VSS branch, or when the parent holds
// a list of it -- a list means the members have identity, because otherwise
// nothing could say which one a value belongs to. Anything else is a value
// object and stays inside its parent's package.
func (m *Model) spawnChildren(root *Def, parent *Package, fam Family, claimed map[string]bool) {
	m.spawnFrom(root, parent, fam, claimed)
}

// spawnFrom promotes the child resources of one type, then recurses into each
// so that a branch nested three deep still becomes addressable.
//
// The depth is the source model's, not a choice made here: a wheel belongs to
// an axle and an axle to a chassis, so `vehicles/{v}/chassis/axles/{a}/wheels/{w}`
// is the name the domain already has.
func (m *Model) spawnFrom(root *Def, parent *Package, fam Family, claimed map[string]bool) {
	for _, f := range root.Fields {
		bt, ok := m.Types[f.Type.Name]
		if !ok || claimed[bt.Name] {
			continue
		}
		// Two rules, by depth.
		//
		// Directly beneath a root, every branch becomes a package: the cabin,
		// the powertrain and the chassis are the top-level areas a consumer
		// addresses, whether or not a vehicle has more than one.
		//
		// Deeper down, only a branch whose members have *identity* is
		// promoted -- a repeated branch, or one the source model gives an
		// instance tag, which is exactly what that tag exists to declare. A
		// singular branch below the top level is a grouping node with one
		// occurrence, so it stays inside its parent's package where its
		// fields can be read in the same call.
		identifiable := f.Type.List || hasInstanceTag(m, bt)
		promote := identifiable
		if parent.Parent == nil {
			promote = isBranch(bt) || identifiable
		}
		if !promote {
			// Not a resource itself, but it may contain one: AmbientLight
			// hangs beneath Cabin.Light, and Light is a grouping node with a
			// single occurrence. Walk through it, attaching anything
			// identifiable to the nearest promoted ancestor.
			//
			// The intermediate name is dropped from the resource name --
			// "vehicles/{v}/cabin/ambientLights/{x}", not ".../light/..." --
			// because a segment with exactly one possible value identifies
			// nothing. The full VSS path survives in the branch annotation's
			// `fqn`, so nothing is lost.
			m.spawnFrom(bt, parent, fam, claimed)
			continue
		}
		dom := parent.Domain
		if dom == "" && fam == FamilyVSS {
			// A direct child of Vehicle: the table names its domain.
			var ok bool
			if dom, ok = domains[bt.Name]; !ok {
				dom = defaultDomain
			}
		}
		pkg := &Package{
			Family: fam, Domain: dom,
			Name: snake(bt.Name), Dir: snake(bt.Name), Root: bt,
			Singular: lowerCamel(bt.Name), Plural: pluralise(lowerCamel(bt.Name)),
			IsList: f.Type.List, Parent: parent,
		}
		if pkg.IsList || identifiable {
			// AIP-123 requires a resource name to alternate collection and
			// identifier. A singleton's name ends in a literal, so a child
			// hung beneath one would put two literals in a row and break the
			// alternation. The name therefore hangs off the nearest ancestor
			// that ends in an identifier -- for a seat, the vehicle rather
			// than the cabin.
			//
			// Nothing is lost by it: the cabin has exactly one occurrence, so
			// a "cabin" segment would identify nothing, and the full VSS path
			// is recorded in the branch annotation's `fqn` either way.
			np := pkg.namingParent()
			pkg.Pattern = np.Pattern + "/" + pkg.Plural + "/{" + snake(bt.Name) + "}"
			pkg.NameParent = np
		} else {
			// The parent holds exactly one, so it is a singleton, AIP-156:
			// its name has no id segment.
			pkg.Pattern = parent.Pattern + "/" + pkg.Singular
			pkg.NameParent = parent
		}
		m.Packages = append(m.Packages, pkg)
		claimed[bt.Name] = true
		pkg.Types = append(pkg.Types, bt)
		// Promote this branch's own children before collecting, so a nested
		// resource is claimed by its own package rather than absorbed here.
		m.spawnFrom(bt, pkg, fam, claimed)
		collect(m, bt, pkg, claimed)
	}
}

// hasInstanceTag reports whether the source model gives a branch an instance
// tag, i.e. declares that its members are individually identifiable.
func hasInstanceTag(m *Model, d *Def) bool {
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
func isBranch(d *Def) bool {
	v, ok := d.Directive("vspec")
	if !ok {
		return false
	}
	e, _ := v.Arg("element")
	return e == "BRANCH" || e == "STRUCT"
}

// collect walks a branch subtree, assigning every type it reaches to pkg.
func collect(m *Model, t *Def, pkg *Package, claimed map[string]bool) {
	if sharedValueTypes[t.Name] {
		// Copied, not claimed: every package referencing it gets its own,
		// which is what keeps each package self-contained under AIP-215.
		for _, have := range pkg.Types {
			if have.Name == t.Name {
				return
			}
		}
		pkg.Types = append(pkg.Types, t)
		for _, f := range t.Fields {
			if e, ok := m.Enums[f.Type.Name]; ok && !isUnitEnum(e.Name) && !isScalarEnum(e.Name) {
				dup := false
				for _, he := range pkg.Enums {
					if he.Name == e.Name {
						dup = true
					}
				}
				if !dup {
					pkg.Enums = append(pkg.Enums, e)
				}
			}
		}
		return
	}
	if claimed[t.Name] {
		// Already placed: either in this package as its root, or in another
		// as a resource of its own. Either way it is not copied here.
		if len(pkg.Types) > 0 && pkg.Types[0].Name == t.Name {
			// The root, already appended by the caller; walk its fields.
		} else {
			return
		}
	} else {
		claimed[t.Name] = true
		pkg.Types = append(pkg.Types, t)
	}
	for _, f := range t.Fields {
		if e, ok := m.Enums[f.Type.Name]; ok && !isUnitEnum(e.Name) && !isScalarEnum(e.Name) {
			if !claimed["enum:"+e.Name] {
				claimed["enum:"+e.Name] = true
				pkg.Enums = append(pkg.Enums, e)
			}
			continue
		}
		if sub, ok := m.Types[f.Type.Name]; ok {
			collect(m, sub, pkg, claimed)
		}
	}
}

// namingParent is the nearest ancestor whose name ends in an identifier.
//
// A singleton's name ends in a literal segment, and AIP-123 forbids two
// literals in a row, so a child of one attaches to the singleton's own parent
// instead -- repeatedly, until an ancestor ending in an identifier is found.
func (p *Package) namingParent() *Package {
	a := p.Parent
	for a != nil && a.Parent != nil && !a.IsList {
		a = a.Parent
	}
	return a
}

// synthDate builds the Date value type, which the source model does not
// declare: it states a calendar date as an ISO 8601 string.
//
// A message rather than that string, because a string cannot say whether
// "03/04" is March or April, and the three components are how every other
// schema in this space models a date -- google.type.Date included.
func synthDate() *Def {
	doc := "Date is a calendar date: a year, a month and a day.\n\n" +
		"The source model states these as ISO 8601 strings. A string cannot be " +
		"validated or compared without parsing it first, and it carries a time " +
		"zone the value does not have -- a production date is the same date " +
		"everywhere. The components are stated instead.\n\n" +
		"Modelled on google.type.Date, which this schema does not use for the " +
		"reason sharedValueTypes gives."
	num := func(name, d string) Field {
		return Field{Name: name, Doc: d, Type: TypeRef{Name: "Int16"}}
	}
	return &Def{
		Kind: "type", Name: "Date", Doc: doc, File: "synthesised",
		Fields: []Field{
			num("year", "Year of the date. Must be from 1 to 9999."),
			num("month", "Month of the year. Must be from 1 to 12."),
			num("day", "Day of the month. Must be from 1 to 31 and valid for the year and month."),
		},
	}
}

// needsDate reports whether any field in the package becomes a Date.
//
// It mirrors the test in assignType rather than sharing it, because that
// function needs a planned field and this runs before planning. The two must
// agree; a disagreement shows up immediately as an unresolved `Date`.
func (p *Package) needsDate() bool {
	for _, d := range p.Types {
		for _, f := range d.Fields {
			if f.Type.Name != "String" {
				continue
			}
			if containsWord(splitWords(f.Name), "date") {
				return true
			}
		}
	}
	return false
}
