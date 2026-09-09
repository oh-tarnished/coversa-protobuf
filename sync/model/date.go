// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package model

// date.go synthesises the one value type neither specification declares.
//
// VSS states a calendar date as a string carrying ISO 8601 text. A string
// cannot be validated or compared without parsing it first, and it implies a
// time zone the value does not have -- a production date is the same date
// everywhere. So a date-valued signal is typed as a Date message instead, and
// that message has to come from somewhere.
//
// google.type.Date is the obvious home and was used, then withdrawn: the
// FlatBuffers and Cap'n Proto generators drop google.* messages other than
// Timestamp and Duration, silently. See docs/decisions.md.

import (
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// placeDates gives a Date message to every package with a date-valued signal.
//
// Reached by *shape* rather than by name: no specification declares a node
// called Date, so nothing places it during the tree walk. A copy per package
// rather than one shared type, because AIP-215 forbids a package from naming
// a message in another one.
func (m *Model) placeDates() {
	for _, pkg := range m.Packages {
		if !pkg.needsDate() || pkg.Holds(dateFQN(pkg)) {
			continue
		}
		pkg.Types = append(pkg.Types, synthDate(pkg))
	}
}

// needsDate reports whether any signal in the package becomes a Date.
//
// It mirrors the test in package plan rather than sharing it, because that
// one needs a planned field and this runs before planning. The two must
// agree; a disagreement shows up immediately as an unresolved `Date`.
func (p *Package) needsDate() bool {
	for _, t := range p.Types {
		for _, c := range t.Children {
			if c.Datatype != "string" {
				continue
			}
			if naming.ContainsWord(naming.SplitWords(c.Name), "date") {
				return true
			}
		}
	}
	return false
}

// dateFQN is the synthetic path the package's Date sits at.
func dateFQN(pkg *Package) string { return pkg.Root.FQN + ".Date" }

// synthDate builds the Date value type for one package.
func synthDate(pkg *Package) *vspec.Node {
	const doc = "Date is a calendar date: a year, a month and a day.\n\n" +
		"The specification states these as ISO 8601 strings. A string cannot be " +
		"validated or compared without parsing it first, and it carries a time " +
		"zone the value does not have -- a production date is the same date " +
		"everywhere. The components are stated instead.\n\n" +
		"Modelled on google.type.Date, which this schema does not use: the " +
		"serialization targets drop it silently. See docs/decisions.md."

	fqn := dateFQN(pkg)
	part := func(name, description, max string) *vspec.Node {
		return &vspec.Node{
			Name: name, FQN: fqn + "." + name,
			Kind: vspec.KindAttribute, Datatype: "uint16",
			Description: description, Min: "1", Max: max,
			File: "synthesised",
		}
	}
	return &vspec.Node{
		Name: "Date", FQN: fqn, Kind: vspec.KindBranch,
		Description: doc, File: "synthesised",
		Children: []*vspec.Node{
			part("Year", "Year of the date.", "9999"),
			part("Month", "Month of the year.", "12"),
			part("Day", "Day of the month, valid for the year and month.", "31"),
		},
	}
}
