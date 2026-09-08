// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// build.go runs the phases that turn indexed definitions into packages:
// indexing and extension folding, then the two family roots, then the
// placement of the synthesised Date.

import (
	"fmt"
	"sort"

	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// Build indexes the definitions, folds `extend type` into its target and
// partitions the specification into packages.
func Build(defs []sdl.Def) (*Model, error) {
	m := &Model{
		Types: map[string]*sdl.Def{},
		Enums: map[string]*sdl.Def{},
	}
	if err := m.index(defs); err != nil {
		return nil, err
	}

	// Date has no source type: the specification states a calendar date as an
	// ISO 8601 string. Synthesised here so a date-valued field has a real
	// message to name.
	m.Types["Date"] = synthDate()

	claimed := map[string]bool{}
	if err := m.buildVSS(claimed); err != nil {
		return nil, err
	}
	if err := m.buildVDM(claimed); err != nil {
		return nil, err
	}
	m.placeDates()
	m.nameEnums()

	sort.Slice(m.Packages, func(i, j int) bool {
		if m.Packages[i].Family != m.Packages[j].Family {
			return m.Packages[i].Family < m.Packages[j].Family
		}
		return m.Packages[i].Name < m.Packages[j].Name
	})
	return m, nil
}

// index records every definition by name and folds extensions into their base.
//
// `extend type Seat { instanceTag: ... }` is how the specification attaches an
// instance tag without editing the domain file. The distinction has no
// protobuf equivalent, so the fields are merged in.
func (m *Model) index(defs []sdl.Def) error {
	var extends []sdl.Def
	for i := range defs {
		d := &defs[i]
		if skipTypes[d.Name] {
			continue
		}
		switch d.Kind {
		case "type":
			if _, dup := m.Types[d.Name]; dup {
				return fmt.Errorf("duplicate type %q", d.Name)
			}
			m.Types[d.Name] = d
		case "enum":
			m.Enums[d.Name] = d
		case "extend type":
			extends = append(extends, *d)
		}
	}
	for _, e := range extends {
		base, ok := m.Types[e.Name]
		if !ok {
			return fmt.Errorf("extend type %q has no base type", e.Name)
		}
		base.Fields = append(base.Fields, e.Fields...)
	}
	return nil
}

// buildVSS creates the vehicle package and everything beneath it.
func (m *Model) buildVSS(claimed map[string]bool) error {
	root, ok := m.Types["Vehicle"]
	if !ok {
		return fmt.Errorf("the specification declares no Vehicle type")
	}

	pkg := &Package{
		Family: FamilyVSS, Name: "vehicle", Dir: "vehicle", Root: root,
		Singular: "vehicle", Plural: "vehicles", Pattern: "vehicles/{vehicle}",
		Types: []*sdl.Def{root},
	}
	claimed["Vehicle"] = true
	for _, f := range root.Fields {
		if e, ok := m.Enums[f.Type.Name]; ok && !IsUnitEnum(e.Name) {
			pkg.Enums = append(pkg.Enums, e)
		}
	}
	m.Packages = append(m.Packages, pkg)
	m.spawn(root, pkg, FamilyVSS, claimed)
	return nil
}

// buildVDM creates a package per resource VDM declares outside the tree.
func (m *Model) buildVDM(claimed map[string]bool) error {
	for _, name := range vdmRoots {
		root, ok := m.Types[name]
		if !ok {
			return fmt.Errorf("the specification declares no %s type", name)
		}
		singular := naming.LowerCamel(name)
		pkg := &Package{
			Family: FamilyVDM, Name: naming.Snake(name), Dir: naming.Snake(name),
			Root: root, Singular: singular, Plural: naming.Pluralise(singular),
		}
		pkg.Pattern = pkg.Plural + "/{" + naming.Snake(name) + "}"
		m.Packages = append(m.Packages, pkg)

		claimed[name] = true
		pkg.Types = append(pkg.Types, root)
		m.spawn(root, pkg, FamilyVDM, claimed)
		m.collect(root, pkg, claimed)
	}
	return nil
}

// placeDates adds the synthesised Date to every package a field will ask for.
//
// Date is reached by *shape*, not by name: the source model types a calendar
// date as a String, and the promotion happens when the field is planned. So
// no package references "Date" by type name and collect never places it.
func (m *Model) placeDates() {
	for _, pkg := range m.Packages {
		if !pkg.needsDate() {
			continue
		}
		if !pkg.holds("Date") {
			pkg.Types = append(pkg.Types, m.Types["Date"])
		}
	}
}

// holds reports whether the package already carries the named type.
func (p *Package) holds(name string) bool {
	for _, d := range p.Types {
		if d.Name == name {
			return true
		}
	}
	return false
}
