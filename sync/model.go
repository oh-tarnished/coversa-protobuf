// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// model.go turns parsed SDL into the package structure the emitter renders:
// one protobuf package per top-level VSS branch, each self-contained.
//
// Self-contained is the load-bearing property. AIP-215 forbids a field from
// referencing a message in another proto package and exempts only `google.*`,
// so a package must hold every type it names. The branch subtrees make this
// possible: across the twenty-odd branches hanging off Vehicle, the only types
// shared between two of them are the unit enums -- and units are carried as an
// annotation rather than a field, so nothing is left to share.

import (
	"fmt"
	"sort"
	"strings"
)

// Model is the whole specification, indexed and partitioned.
type Model struct {
	// Spec is the revision pin every generated banner names.
	Spec     Spec
	Types    map[string]*Def // object types by name
	Enums    map[string]*Def // enum types by name
	Packages []*Package      // one per top-level branch, plus vehicle itself
}

// Family is the specification a package comes from: the VSS vehicle tree, or
// the VDM types defined outside it.
type Family string

const (
	// FamilyVSS is the COVESA Vehicle Signal Specification tree under Vehicle.
	FamilyVSS Family = "vss"
	// FamilyVDM is everything VDM defines outside that tree -- the people,
	// places and events a connected vehicle interacts with.
	FamilyVDM Family = "vdm"
)

// Package is one emitted protobuf package.
type Package struct {
	Family   Family   // which specification it comes from
	Domain   string   // functional area, e.g. "interior"; empty at a root
	Name     string   // proto package leaf, e.g. "current_location"
	Dir      string   // directory leaf, same as Name
	Root     *Def     // the branch type that is this package's resource
	Types    []*Def   // every object type in the package, Root first
	Enums    []*Def   // every enum in the package
	Singular string   // AIP singular, e.g. "cabin"
	Plural   string   // AIP plural, e.g. "cabins"
	Pattern  string   // resource name pattern
	IsList   bool     // the branch is a repeated field on its parent
	Parent   *Package // the resource this one hangs beneath, nil at the root
	// NameParent is the resource this one's *name* hangs beneath, which
	// differs from Parent when an intervening ancestor is a singleton.
	NameParent *Package
	Imports    []string // extra well-known imports the emitted files need
}

// unitEnumSuffix marks the quantity-kind enums, which are not emitted as types
// -- they become the `unit` annotation instead.
const unitEnumSuffix = "UnitEnum"

// BuildModel indexes the definitions, folds `extend type` into its target and
// partitions the specification into packages.
func BuildModel(defs []Def) (*Model, error) {
	m := &Model{Types: map[string]*Def{}, Enums: map[string]*Def{}}
	var extends []Def
	for i := range defs {
		d := &defs[i]
		switch d.Kind {
		case "type":
			if skipTypes[d.Name] {
				continue
			}
			if _, dup := m.Types[d.Name]; dup {
				return nil, fmt.Errorf("duplicate type %q", d.Name)
			}
			m.Types[d.Name] = d
		case "enum":
			m.Enums[d.Name] = d
		case "extend type":
			if skipTypes[d.Name] {
				continue
			}
			extends = append(extends, *d)
		}
	}
	for _, e := range extends {
		t, ok := m.Types[e.Name]
		if !ok {
			return nil, fmt.Errorf("extend type %q has no base type", e.Name)
		}
		t.Fields = append(t.Fields, e.Fields...)
	}

	claimed := map[string]bool{}

	// Date has no source type: the specification states a calendar date as an
	// ISO 8601 string. It is synthesised here so a date-valued field has a
	// real message to name, with the same three components google.type.Date
	// uses -- the model that was withdrawn, see sharedValueTypes.
	m.Types["Date"] = synthDate()

	// The vehicle tree. Every branch hanging off Vehicle becomes a package.
	root, ok := m.Types["Vehicle"]
	if !ok {
		return nil, fmt.Errorf("the specification declares no Vehicle type")
	}
	vp := &Package{
		Family: FamilyVSS, Name: "vehicle", Dir: "vehicle", Root: root,
		Singular: "vehicle", Plural: "vehicles", Pattern: "vehicles/{vehicle}",
		Types: []*Def{root},
	}
	claimed["Vehicle"] = true
	for _, f := range root.Fields {
		if e, ok := m.Enums[f.Type.Name]; ok && !isUnitEnum(e.Name) {
			vp.Enums = append(vp.Enums, e)
		}
	}
	m.Packages = append(m.Packages, vp)
	m.spawnChildren(root, vp, FamilyVSS, claimed)

	// Everything VDM declares outside the vehicle tree.
	for _, name := range vdmRoots {
		rt, ok := m.Types[name]
		if !ok {
			return nil, fmt.Errorf("the specification declares no %s type", name)
		}
		pkg := &Package{
			Family: FamilyVDM, Name: snake(name), Dir: snake(name), Root: rt,
			Singular: lowerCamel(name), Plural: pluralise(lowerCamel(name)),
		}
		pkg.Pattern = pkg.Plural + "/{" + snake(name) + "}"
		m.Packages = append(m.Packages, pkg)
		claimed[name] = true
		pkg.Types = append(pkg.Types, rt)
		m.spawnChildren(rt, pkg, FamilyVDM, claimed)
		// Value objects the root embeds stay inside its package.
		collect(m, rt, pkg, claimed)
	}

	// Date is reached by *shape*, not by name: the source model types a
	// calendar date as a String, and the promotion to a Date message happens
	// when the field is planned. So no package references "Date" by type name
	// and collect never places it. Add it wherever a field will ask for it.
	for _, pkg := range m.Packages {
		if !pkg.needsDate() {
			continue
		}
		has := false
		for _, d := range pkg.Types {
			if d.Name == "Date" {
				has = true
			}
		}
		if !has {
			pkg.Types = append(pkg.Types, m.Types["Date"])
		}
	}

	sort.Slice(m.Packages, func(i, j int) bool {
		if m.Packages[i].Family != m.Packages[j].Family {
			return m.Packages[i].Family < m.Packages[j].Family
		}
		return m.Packages[i].Name < m.Packages[j].Name
	})
	return m, nil
}

// isUnitEnum reports whether an enum is one of the quantity-kind unit enums,
// which become annotations rather than emitted types.
func isUnitEnum(name string) bool { return strings.HasSuffix(name, unitEnumSuffix) }

// isScalarEnum reports whether an enum is replaced by a documented scalar and
// therefore never emitted. See scalarEnums.
func isScalarEnum(name string) bool { _, ok := scalarEnums[name]; return ok }

// packageOf returns the package whose resource is the named type, or nil when
// the type is not a resource root.
func (m *Model) packageOf(name string) *Package {
	for _, p := range m.Packages {
		if p.Root.Name == name {
			return p
		}
	}
	return nil
}
