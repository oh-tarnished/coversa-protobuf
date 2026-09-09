// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package main

// survey.go reports what the generator parsed: the two revisions, the shape
// of the tree, and anything filed by fallback rather than by decision.
//
// Run it after a specification bump. `buf breaking` catches a field that
// moved; nothing else reports a branch that landed in a domain because the
// table did not name it, which is a decision nobody made.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// surveyModel prints the parsed model and its naming traps.
func surveyModel(m *model.Model) error {
	fmt.Printf("vss %s (%s), vdm %s (%s)\n\n",
		m.Spec.VSS.Version, m.Spec.VSS.Short(),
		m.Spec.VDM.Version, m.Spec.VDM.Short())

	kinds := map[vspec.Kind]int{}
	for _, root := range m.Roots {
		root.Walk(func(n *vspec.Node) { kinds[n.Kind]++ })
	}
	for _, k := range []vspec.Kind{vspec.KindBranch, vspec.KindSensor,
		vspec.KindActuator, vspec.KindAttribute, vspec.KindStruct, vspec.KindProperty} {
		if kinds[k] > 0 {
			fmt.Printf("%-12s %d\n", k, kinds[k])
		}
	}
	fmt.Printf("%-12s %d\n%-12s %d\n", "packages", len(m.Packages), "units", len(m.Units.Units))

	surveyDomains(m)
	surveyTraps(m)
	return nil
}

// surveyDomains reports the top-level branches the domain table does not name.
//
// One lands in platform, which is a fallback rather than a decision. Reported
// so a VSS release adding a branch is visible rather than silently filed --
// Orientation and Safety both arrived this way.
func surveyDomains(m *model.Model) {
	var unnamed []string
	for _, p := range m.Packages {
		if p.Family == model.FamilyVSS && p.Parent != nil &&
			p.Parent.Parent == nil && !model.DomainNamed(p.Root.Name) {
			unnamed = append(unnamed, p.Root.Name)
		}
	}
	if len(unnamed) == 0 {
		return
	}
	sort.Strings(unnamed)
	fmt.Printf("\ntop-level branches with no domain (filed under platform): %s\n",
		strings.Join(unnamed, ", "))
}

// surveyTraps prints the naming traps: resolved ones counted, unresolved ones
// listed with what to do about each.
func surveyTraps(m *model.Model) {
	traps := findTraps(m)

	var open []trap
	for _, t := range traps {
		if !t.resolved {
			open = append(open, t)
		}
	}

	fmt.Printf("\nAIP naming traps: %d found, %d resolved\n",
		len(traps), len(traps)-len(open))
	if len(open) == 0 {
		return
	}
	fmt.Printf("\n%d unresolved. Add a rename to catalog.FieldRenames, or a reason to\n"+
		"catalog.AcceptedTraps, and a row to the catalogue in docs/conventions.md:\n\n",
		len(open))
	for _, t := range open {
		fmt.Printf("  %-46s %-24s %s\n", t.owner, t.field, t.rule)
	}
}
