// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package model

// resource.go answers what shape a package's API surface takes: singleton or
// collection, and the REST paths that follow from its resource name.

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/catalog"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
)

// MessageName renders a source object type as a protobuf message name.
func MessageName(g string) string {
	if r, ok := catalog.TypeRenames[g]; ok {
		return r
	}
	return naming.Pascal(g)
}

// ResourceName is the message name of this package's resource.
func (p *Package) ResourceName() string { return MessageName(p.Root.Name) }

// PluralType renders the plural as a type-cased name, e.g. "ControlUnits".
func (p *Package) PluralType() string { return naming.Pascal(p.Plural) }

// ServiceName is the collection name the service is called by, AIP-131.
//
// Where a branch's plural equals its singular -- Adas, Diagnostics,
// Connectivity, all mass nouns or already-plural names -- the collection name
// would collide with the message name in the same package, which protobuf
// rejects. Those services take a `Service` suffix: the collection name is
// still the resource's own plural, so AIP-123's tie between plural and URL
// segment is untouched, and only the gRPC service identifier differs.
func (p *Package) ServiceName() string {
	if p.PluralType() == p.ResourceName() {
		return p.PluralType() + "Service"
	}
	return p.PluralType()
}

// TopLevel reports whether the resource has no parent resource above it.
func (p *Package) TopLevel() bool { return p.Parent == nil }

// Singleton reports whether the parent holds exactly one of this branch, in
// which case AIP-156 governs and there is no Create, Delete or List.
//
// Not a reduced CRUD surface: AIP-156 defines the complete method set for a
// singleton, and Create and Delete are not in it. Nothing can make a second
// cabin or remove the only one.
func (p *Package) Singleton() bool { return p.Parent != nil && !p.IsList }

// HTTPPath renders the resource's REST path from its own pattern.
//
// Derived rather than assembled from the family: a ChargingPoint hangs
// beneath a ChargingStation, not beneath a vehicle, and AIP-127 checks the
// HTTP template against the resource's declared pattern.
func (p *Package) HTTPPath() string {
	return "/v1/{name=" + patternToPath(p.Pattern) + "}"
}

// CollectionPath renders the collection's REST path, for List and Create.
//
// A root resource's collection is addressed directly; a child's hangs beneath
// its parent, whose name becomes the `parent` variable.
func (p *Package) CollectionPath() string {
	if p.TopLevel() {
		return "/v1/" + p.Plural
	}
	return "/v1/{parent=" + patternToPath(p.NameParent.Pattern) + "}/" + p.Plural
}

// ParentName is the message name of the resource this one's name hangs
// beneath, or "" at a root.
func (p *Package) ParentName() string {
	if p.NameParent == nil {
		return ""
	}
	return MessageName(p.NameParent.Root.Name)
}

// patternToPath turns a resource name pattern into an HTTP template body:
// every `{segment}` becomes a `*` wildcard.
func patternToPath(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, "{") {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, "/")
}

// PatternRegex turns a resource name pattern into the anchored regex that
// validates it: every `{segment}` becomes one path component.
func PatternRegex(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, "{") {
			parts[i] = "[^/]+"
		}
	}
	return "^" + strings.Join(parts, "/") + "$"
}
