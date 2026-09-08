// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// collect.go assigns the types a promoted branch reaches to its package, and
// synthesises the one value type the source model does not declare.

import (
	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// collect walks a branch subtree, assigning every value object it reaches to
// pkg. Types already promoted to packages of their own are left alone.
func (m *Model) collect(t *sdl.Def, pkg *Package, claimed map[string]bool) {
	if sharedValueTypes[t.Name] {
		m.copyShared(t, pkg)
		return
	}

	switch {
	case !claimed[t.Name]:
		claimed[t.Name] = true
		pkg.Types = append(pkg.Types, t)
	case len(pkg.Types) > 0 && pkg.Types[0].Name == t.Name:
		// The package's own root, appended by the caller. Fall through to
		// walk its fields.
	default:
		// Placed in another package already; not copied here.
		return
	}

	for _, f := range t.Fields {
		if e, ok := m.Enums[f.Type.Name]; ok && m.emitsEnum(e.Name) {
			if !claimed["enum:"+e.Name] {
				claimed["enum:"+e.Name] = true
				pkg.Enums = append(pkg.Enums, e)
			}
			continue
		}
		if sub, ok := m.Types[f.Type.Name]; ok {
			m.collect(sub, pkg, claimed)
		}
	}
}

// copyShared gives the package its own copy of a shared value type.
//
// Copied, not claimed: every package referencing it gets one, which is what
// keeps each package self-contained under AIP-215.
func (m *Model) copyShared(t *sdl.Def, pkg *Package) {
	if pkg.holds(t.Name) {
		return
	}
	pkg.Types = append(pkg.Types, t)

	for _, f := range t.Fields {
		e, ok := m.Enums[f.Type.Name]
		if !ok || !m.emitsEnum(e.Name) || pkg.holdsEnum(e.Name) {
			continue
		}
		pkg.Enums = append(pkg.Enums, e)
	}
}

// emitsEnum reports whether an enum becomes a protobuf enum, rather than an
// annotation (unit enums) or a documented scalar (country codes).
func (m *Model) emitsEnum(name string) bool {
	return !IsUnitEnum(name) && !catalog.IsScalarEnum(name)
}

// holdsEnum reports whether the package already carries the named enum.
func (p *Package) holdsEnum(name string) bool {
	for _, e := range p.Enums {
		if e.Name == name {
			return true
		}
	}
	return false
}

// needsDate reports whether any field in the package becomes a Date.
//
// It mirrors the test in the plan package rather than sharing it, because
// that one needs a planned field and this runs before planning. The two must
// agree; a disagreement shows up immediately as an unresolved `Date`.
func (p *Package) needsDate() bool {
	for _, d := range p.Types {
		for _, f := range d.Fields {
			if f.Type.Name != "String" {
				continue
			}
			if naming.ContainsWord(naming.SplitWords(f.Name), "date") {
				return true
			}
		}
	}
	return false
}

// synthDate builds the Date value type, which the source model does not
// declare: it states a calendar date as an ISO 8601 string.
//
// A message rather than that string, because a string cannot be validated or
// compared without parsing it first, and it carries a time zone the value
// does not have -- a production date is the same date everywhere.
func synthDate() *sdl.Def {
	const doc = "Date is a calendar date: a year, a month and a day.\n\n" +
		"The source model states these as ISO 8601 strings. A string cannot be " +
		"validated or compared without parsing it first, and it carries a time " +
		"zone the value does not have -- a production date is the same date " +
		"everywhere. The components are stated instead.\n\n" +
		"Modelled on google.type.Date, which this schema does not use for the " +
		"reason sharedValueTypes gives."

	part := func(name, d string) sdl.Field {
		return sdl.Field{Name: name, Doc: d, Type: sdl.TypeRef{Name: "Int16"}}
	}
	return &sdl.Def{
		Kind: "type", Name: "Date", Doc: doc, File: "synthesised",
		Fields: []sdl.Field{
			part("year", "Year of the date. Must be from 1 to 9999."),
			part("month", "Month of the year. Must be from 1 to 12."),
			part("day", "Day of the month. Must be from 1 to 31 and valid for the year and month."),
		},
	}
}
