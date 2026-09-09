// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// collect.go assigns the embedded messages a promoted branch reaches to its
// package.

import "github.com/oh-tarnished/coversa-protobuf/sync/vspec"

// collect walks a branch subtree, assigning every grouping node that was not
// promoted to pkg as an embedded message.
func (m *Model) collect(parent *vspec.Node, pkg *Package, claimed map[string]bool) {
	for _, child := range parent.Children {
		if child.Kind != vspec.KindBranch || child.Ref != "" {
			continue
		}
		if claimed[child.FQN] {
			// Promoted to a package of its own; not copied here.
			continue
		}
		claimed[child.FQN] = true
		pkg.Types = append(pkg.Types, child)
		m.collect(child, pkg, claimed)
	}
}

// Signals returns the value-carrying children of a message, in specification
// order.
//
// A branch child is a nested message and a reference is an association;
// neither is a signal. Everything else carries a value.
func Signals(n *vspec.Node) []*vspec.Node {
	var out []*vspec.Node
	for _, c := range n.Children {
		if c.Kind == vspec.KindBranch && c.Ref == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Embedded returns the children of a message that are messages themselves.
func Embedded(n *vspec.Node) []*vspec.Node {
	var out []*vspec.Node
	for _, c := range n.Children {
		if c.Kind == vspec.KindBranch && c.Ref == "" {
			out = append(out, c)
		}
	}
	return out
}
