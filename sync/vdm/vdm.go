// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package vdm reads what COVESA's Vehicle Data Model defines outside the
// vehicle, and expresses it in the same shape VSS is read into.
//
// # Why a translation, and which way it runs
//
// VDM is written in GraphQL SDL and describes two different things: a
// translation of the Vehicle Signal Specification, and a handful of resources
// VSS has no concept of -- a person, a charging station, a charging session.
//
// The first is not read at all. It is lossy, and the specification it
// translates is available directly; see package vspec.
//
// The second is read here and converted *into* the vspec node shape, so
// everything downstream works on one representation. The direction matters: a
// year ago this generator translated the other way, and every VSS-only fact
// had nowhere to land.
//
// # What is lost, and why nothing
//
// GraphQL says less than VSS, not more: no units, no bounds, no signal kind.
// A converted node therefore carries a description and a type and stops
// there, which is exactly what these resources have to say.
package vdm

import (
	"fmt"

	"github.com/the-protobuf-project/vdm/sync/sdl"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// Roots are the resources VDM declares outside the vehicle tree, in the order
// they are emitted.
var Roots = []string{"Person", "ChargingStation", "ChargingSession"}

// references are the types a field points at rather than inlines.
//
// The three roots, plus the vehicle -- which is not read from GraphQL at all
// any more, so inlining it would embed a stale translation of a tree the
// schema already has from VSS.
var references = map[string]bool{
	"Vehicle": true, "Person": true,
	"ChargingStation": true, "ChargingSession": true,
	"ChargingPoint": true,
}

// owners names the one place a referenced type is *contained* rather than
// pointed at.
//
// A charging point is both: the station it stands at holds it -- its resource
// name extends the station's -- while a charging session merely names the
// point it used. Same type, two relationships, and only the containment may
// inline. Without this the point would be copied into the session, which
// would say a session *holds* a connector.
//
// A table rather than a rule because VDM declares four resources and the
// relationships are known; a rule inferred from field cardinality would guess
// wrongly the moment a second such type appeared.
var owners = map[string]string{"ChargingPoint": "ChargingStation"}

// scalars maps GraphQL's built-in types onto VSS datatype names, so a
// converted field is indistinguishable from one the specification declared.
var scalars = map[string]string{
	"String": "string", "ID": "string", "Boolean": "boolean",
	"Float": "double", "Int": "int32",
	"Int8": "int8", "UInt8": "uint8", "Int16": "int16", "UInt16": "uint16",
	"UInt32": "uint32", "Int64": "int64", "UInt64": "uint64",
}

// Convert turns the SDL definitions into vspec nodes rooted at each entry of
// Roots.
//
// A field naming another object type becomes a child branch with that type's
// own children inlined, because vspec has no notion of a type reference: a
// branch's children *are* its fields. Inlining a type used twice yields two
// copies, which is what the schema wants anyway -- AIP-215 forbids a package
// from naming a message in another one.
func Convert(defs []sdl.Def) ([]*vspec.Node, error) {
	types, enums := index(defs)

	out := make([]*vspec.Node, 0, len(Roots))
	for _, name := range Roots {
		def, ok := types[name]
		if !ok {
			return nil, fmt.Errorf("the model declares no %s type", name)
		}
		node, err := convert(name, name, name, def, types, enums, map[string]bool{})
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

// index sorts the definitions into object types and enums, folding each
// `extend type` into its base.
func index(defs []sdl.Def) (map[string]*sdl.Def, map[string]*sdl.Def) {
	types, enums := map[string]*sdl.Def{}, map[string]*sdl.Def{}

	var extends []sdl.Def
	for i := range defs {
		d := &defs[i]
		switch d.Kind {
		case "type":
			types[d.Name] = d
		case "enum":
			enums[d.Name] = d
		case "extend type":
			extends = append(extends, *d)
		}
	}
	for _, e := range extends {
		if base, ok := types[e.Name]; ok {
			base.Fields = append(base.Fields, e.Fields...)
		}
	}
	return types, enums
}
