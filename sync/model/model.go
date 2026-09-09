// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package model turns the parsed specifications into the package structure
// the emitters render: one protobuf package per resource, each self-contained.
//
// # One tree, two sources
//
// The vehicle comes from the Vehicle Signal Specification and everything
// around it from the Vehicle Data Model, but both arrive as [vspec.Node]
// trees — the VDM half is converted on the way in, by package vdm. Nothing
// below this point knows which specification a node came from, which is what
// keeps the partitioning rules from having two cases everywhere.
//
// # Self-contained is the load-bearing property
//
// AIP-215 forbids a field from referencing a message in another proto package
// and exempts only `google.*`, so a package must hold every type it names.
// The branch subtrees make that possible: across the branches hanging off
// Vehicle, the only things shared between two of them are units — and a unit
// is an annotation rather than a field, so nothing is left to share.
package model

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/spec"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// Family is the specification a package comes from.
type Family string

const (
	// FamilyVSS is the Vehicle Signal Specification tree under Vehicle.
	FamilyVSS Family = "vss"

	// FamilyVDM is everything the Vehicle Data Model defines outside that
	// tree — the people, places and events a connected vehicle interacts
	// with.
	FamilyVDM Family = "vdm"
)

// Model is the whole specification, indexed and partitioned.
type Model struct {
	// Spec is the revision pin every generated banner names.
	Spec spec.Spec

	// Units is the unit and quantity vocabulary VSS ships beside the tree.
	Units *vspec.Catalogue

	// Roots are the trees the packages were built from: the vehicle first,
	// then the VDM resources.
	Roots []*vspec.Node

	// Nodes indexes every node by fully qualified name.
	Nodes map[string]*vspec.Node

	// Packages is one entry per resource, sorted by family then name.
	Packages []*Package

	// enumNames maps a node's fqn to the protobuf enum name emitted for its
	// allowed-value set. Resolved per package once the packages are known.
	enumNames map[string]string
}

// Package is one emitted protobuf package.
type Package struct {
	Family Family // which specification it comes from
	Domain string // functional area, e.g. "interior"; empty at a root
	Name   string // proto package leaf, e.g. "current_location"
	Dir    string // directory leaf, same as Name

	Root  *vspec.Node   // the branch that is this package's resource
	Types []*vspec.Node // every message in the package, Root first

	Singular string // AIP singular, e.g. "cabin"
	Plural   string // AIP plural, e.g. "cabins"
	Pattern  string // resource name pattern
	IsList   bool   // the parent holds many of this branch

	// Parent is the resource this one hangs beneath, nil at a root.
	Parent *Package

	// NameParent is the resource this one's *name* hangs beneath, which
	// differs from Parent when an intervening ancestor is a singleton.
	NameParent *Package

	// qualified is the name that separates this package from others sharing
	// its leaf, empty where the leaf is already unique. Held because the
	// directory always takes it and the collection segment only sometimes
	// does. See disambiguate.go.
	qualified string
}

// Enums returns the nodes in the package whose allowed values become a
// protobuf enum, in declaration order.
//
// Derived rather than stored: an enum belongs to the signal that constrains
// it, so the signal is the record and a second list would be a second thing
// to keep in step.
func (p *Package) Enums() []*vspec.Node {
	var out []*vspec.Node
	for _, t := range p.Types {
		for _, c := range t.Children {
			if HasEnum(c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// HasEnum reports whether a node's values become a protobuf enum.
//
// VSS states allowed values two ways: `allowed`, a list of names, and `enum`,
// a mapping carrying the specification's own numbering. Either makes an enum;
// a signal with neither is a plain scalar.
//
// A set the catalogue replaces with a documented scalar is not one. AIP-143
// wants a country code as a string, and Cap'n Proto cannot represent the enum
// at all — COUNTRY_CODE_AS and COUNTRY_CODE_IN lose the prefix and land on
// reserved words. Answered here so the type is never emitted and never named.
func HasEnum(n *vspec.Node) bool {
	if catalog.IsScalarEnum(n.EnumName) {
		return false
	}
	return len(n.Allowed) > 0 || len(n.Enum) > 0
}

// Holds reports whether the package carries the named message.
func (p *Package) Holds(fqn string) bool {
	for _, t := range p.Types {
		if t.FQN == fqn {
			return true
		}
	}
	return false
}

// PackageOf returns the package whose resource is the named node, or nil.
func (m *Model) PackageOf(fqn string) *Package {
	for _, p := range m.Packages {
		if p.Root.FQN == fqn {
			return p
		}
	}
	return nil
}

// EnumName is the protobuf name emitted for a node's allowed-value set.
func (m *Model) EnumName(fqn string) string {
	if n, ok := m.enumNames[fqn]; ok {
		return n
	}
	return ""
}

// leaf is the last segment of a dotted path.
func leaf(fqn string) string {
	if i := strings.LastIndex(fqn, "."); i >= 0 {
		return fqn[i+1:]
	}
	return fqn
}
