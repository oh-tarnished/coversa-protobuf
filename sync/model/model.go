// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package model turns parsed SDL into the package structure the emitters
// render: one protobuf package per resource, each self-contained.
//
// Self-contained is the load-bearing property. AIP-215 forbids a field from
// referencing a message in another proto package and exempts only `google.*`,
// so a package must hold every type it names. The branch subtrees make that
// possible: across the branches hanging off Vehicle, the only types shared
// between two of them are the unit enums -- and units are carried as an
// annotation rather than a field, so nothing is left to share.
package model

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/sdl"
	"github.com/the-protobuf-project/vdm/sync/spec"
)

// Family is the specification a package comes from.
type Family string

const (
	// FamilyVSS is the COVESA Vehicle Signal Specification tree under Vehicle.
	FamilyVSS Family = "vss"

	// FamilyVDM is everything VDM defines outside that tree -- the people,
	// places and events a connected vehicle interacts with.
	FamilyVDM Family = "vdm"
)

// unitEnumSuffix marks the quantity-kind enums, which become the `unit`
// annotation rather than emitted types.
const unitEnumSuffix = "UnitEnum"

// Model is the whole specification, indexed and partitioned.
type Model struct {
	// Spec is the revision pin every generated banner names.
	Spec spec.Spec

	// Types and Enums index every definition by name.
	Types map[string]*sdl.Def
	Enums map[string]*sdl.Def

	// Packages is one entry per resource, sorted by family then name.
	Packages []*Package

	// enumNames maps a source enum name to the name emitted for it. Resolved
	// per package once the packages are known; see enums.go.
	enumNames map[string]string
}

// Package is one emitted protobuf package.
type Package struct {
	Family Family // which specification it comes from
	Domain string // functional area, e.g. "interior"; empty at a root
	Name   string // proto package leaf, e.g. "current_location"
	Dir    string // directory leaf, same as Name

	Root  *sdl.Def   // the branch type that is this package's resource
	Types []*sdl.Def // every object type in the package, Root first
	Enums []*sdl.Def // every enum in the package

	Singular string // AIP singular, e.g. "cabin"
	Plural   string // AIP plural, e.g. "cabins"
	Pattern  string // resource name pattern
	IsList   bool   // the branch is a repeated field on its parent

	// Parent is the resource this one hangs beneath, nil at a root.
	Parent *Package

	// NameParent is the resource this one's *name* hangs beneath, which
	// differs from Parent when an intervening ancestor is a singleton. See
	// namingParent.
	NameParent *Package
}

// IsUnitEnum reports whether an enum is one of the quantity-kind unit enums,
// which become annotations rather than emitted types.
func IsUnitEnum(name string) bool { return strings.HasSuffix(name, unitEnumSuffix) }

// PackageOf returns the package whose resource is the named type, or nil when
// the type is not a resource root.
func (m *Model) PackageOf(name string) *Package {
	for _, p := range m.Packages {
		if p.Root.Name == name {
			return p
		}
	}
	return nil
}

// Holds reports whether the package carries the named object type.
func (p *Package) Holds(name string) bool { return p.holds(name) }
