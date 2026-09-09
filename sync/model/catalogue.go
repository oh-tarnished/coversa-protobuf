// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package model

// catalogue.go checks the resolution tables against the tree they resolve.
//
// Every entry in package catalog names a signal by its fully qualified name.
// A specification bump can remove that signal, and the entry then resolves
// nothing -- it sits in the table, and in the catalogue at the foot of
// docs/conventions.md, describing a decision about a field that no longer
// exists. Nothing else can see that: the rename is simply never applied, the
// output is valid, and every linter passes.
//
// So it is a build error, per rule 15. The generator fails loudly or not at
// all, and a bump that drops a signal has to say so.
//
// Called from package load rather than from [Build], because it is a check
// against the *whole* specification: a fixture tree legitimately contains
// almost none of these signals, and running it there would only report that
// the fixture is small.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/catalog"
)

// CheckCatalogue reports every resolution table entry naming no node.
func (m *Model) CheckCatalogue() error {
	var stale []string

	for fqn, to := range catalog.FieldRenames {
		if _, ok := m.Nodes[fqn]; !ok {
			stale = append(stale, fmt.Sprintf("FieldRenames[%q] -> %q", fqn, to))
		}
	}
	for fqn := range catalog.AcceptedTraps {
		if _, ok := m.Nodes[fqn]; !ok {
			stale = append(stale, fmt.Sprintf("AcceptedTraps[%q]", fqn))
		}
	}
	for name, table := range map[string]map[string]bool{
		"DurationFields": catalog.DurationFields,
		"RoundTripIDs":   catalog.RoundTripIDs,
	} {
		for fqn := range table {
			if _, ok := m.Nodes[fqn]; !ok {
				stale = append(stale, fmt.Sprintf("%s[%q]", name, fqn))
			}
		}
	}
	if len(stale) == 0 {
		return nil
	}

	sort.Strings(stale)
	return fmt.Errorf("%d catalogue entries name a signal the specification no "+
		"longer has:\n  %s\n\nRemove each from sync/catalog and from the catalogue "+
		"at the foot of docs/conventions.md, and record the removal in docs/spec.md",
		len(stale), strings.Join(stale, "\n  "))
}

// checkUnique reports two packages that would write over each other.
//
// A proto package is a directory, so two packages sharing one means the
// second writes `axle.proto` over the first and only the last survives -- no
// error, valid output, four resources simply gone. A resource pattern is an
// address, so two packages sharing one makes `vehicles/{v}/axles/{a}` name
// more than one resource, which AIP-123 forbids and no linter here can see,
// because the losing package was never emitted for it to check.
//
// [Model.disambiguate] resolves both. This is the assertion that it did.
func (m *Model) checkUnique() error {
	for _, c := range []struct {
		what string
		key  func(*Package) string
	}{
		{"proto package", func(p *Package) string { return p.ProtoPackage() }},
		{"resource pattern", func(p *Package) string { return p.Pattern }},
	} {
		seen := map[string]*Package{}
		for _, p := range m.Packages {
			k := c.key(p)
			if first, dup := seen[k]; dup {
				return fmt.Errorf("%s %q is claimed by both %s and %s; "+
					"disambiguate did not separate them",
					c.what, k, first.Root.FQN, p.Root.FQN)
			}
			seen[k] = p
		}
	}
	return nil
}
