// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// build.go runs the phases that turn node trees into packages: indexing,
// then a package per resource, then the names of the enums they carry.

import (
	"fmt"
	"sort"

	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// Build partitions the specification trees into packages.
//
// vehicle is the tree read from VSS; others are the resources the Vehicle
// Data Model declares outside it, already converted to the same shape.
func Build(vehicle *vspec.Node, others []*vspec.Node, units *vspec.Catalogue) (*Model, error) {
	m := &Model{
		Units: units,
		Roots: append([]*vspec.Node{vehicle}, others...),
		Nodes: map[string]*vspec.Node{},
	}
	for _, root := range m.Roots {
		root.Walk(func(n *vspec.Node) { m.Nodes[n.FQN] = n })
	}

	if err := m.buildRoot(vehicle, FamilyVSS); err != nil {
		return nil, err
	}
	for _, root := range others {
		if err := m.buildRoot(root, FamilyVDM); err != nil {
			return nil, err
		}
	}

	m.disambiguate()
	if err := m.checkUnique(); err != nil {
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

// buildRoot creates the package for one tree's root and everything under it.
func (m *Model) buildRoot(root *vspec.Node, fam Family) error {
	if root.Kind != vspec.KindBranch {
		return fmt.Errorf("%s is a %s, not a branch; only a branch can be a resource", root.FQN, root.Kind)
	}

	singular := naming.LowerCamel(root.Name)
	pkg := &Package{
		Family: fam,
		Name:   naming.Snake(root.Name), Dir: naming.Snake(root.Name),
		Root: root, Types: []*vspec.Node{root},
		Singular: singular, Plural: naming.Pluralise(singular),
	}
	pkg.Pattern = pkg.Plural + "/{" + pkg.Singular + "}"
	m.Packages = append(m.Packages, pkg)

	claimed := map[string]bool{root.FQN: true}
	m.spawn(root, pkg, fam, claimed)
	m.collect(root, pkg, claimed)
	return nil
}
