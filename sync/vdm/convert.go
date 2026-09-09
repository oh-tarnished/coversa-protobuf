// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vdm

// convert.go turns one GraphQL object type, and everything it references,
// into a vspec branch.

import (
	"fmt"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/sdl"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// convert builds the branch for one object type.
//
// seen guards against a type that references itself, directly or through a
// cycle. VDM has none today; inlining one would not terminate, so it is an
// error rather than a hang.
func convert(root, name, fqn string, def *sdl.Def, types, enums map[string]*sdl.Def, seen map[string]bool) (*vspec.Node, error) {
	if seen[def.Name] {
		return nil, fmt.Errorf("%s refers to itself; vspec has no way to express a cycle", def.Name)
	}
	seen[def.Name] = true
	defer delete(seen, def.Name)

	node := &vspec.Node{
		Name:        name,
		FQN:         fqn,
		Kind:        vspec.KindBranch,
		Description: strings.Join(strings.Fields(def.Doc), " "),
		File:        def.File,
	}

	for _, f := range def.Fields {
		child, err := field(f, root, fqn, types, enums, seen)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", def.Name, f.Name, err)
		}
		node.Children = append(node.Children, child)
	}
	return node, nil
}

// field converts one GraphQL field.
func field(f sdl.Field, root, parent string, types, enums map[string]*sdl.Def, seen map[string]bool) (*vspec.Node, error) {
	fqn := parent + "." + upperFirst(f.Name)

	// A field naming a resource points at it rather than containing it.
	// Inlining would copy an entire tree -- the Vehicle alone is 795 nodes --
	// and would say a session *holds* a vehicle, which it does not.
	if references[f.Type.Name] && owners[f.Type.Name] != root {
		return &vspec.Node{
			Name:        f.Name,
			FQN:         fqn,
			Kind:        vspec.KindActuator,
			Datatype:    "string",
			Description: strings.Join(strings.Fields(f.Doc), " "),
			Ref:         f.Type.Name,
			Repeated:    f.Type.List,
			Required:    f.Type.NonNull,
		}, nil
	}

	// Any other object-typed field becomes a branch with that type inlined.
	if def, ok := types[f.Type.Name]; ok {
		child, err := convert(root, f.Name, fqn, def, types, enums, seen)
		if err != nil {
			return nil, err
		}
		child.Repeated = f.Type.List
		child.Required = f.Type.NonNull
		return child, nil
	}

	node := &vspec.Node{
		Name:        f.Name,
		FQN:         fqn,
		Description: strings.Join(strings.Fields(f.Doc), " "),
		Repeated:    f.Type.List,
		Required:    f.Type.NonNull,
	}

	// GraphQL states no signal kind. These resources are records a client
	// creates and edits rather than values a vehicle reports, so every field
	// is writable -- which is what actuator means downstream.
	node.Kind = vspec.KindActuator

	if def, ok := enums[f.Type.Name]; ok {
		node.Datatype = "string"
		node.EnumName = f.Type.Name
		for _, v := range def.Values {
			node.Allowed = append(node.Allowed, v.Name)
		}
		return node, nil
	}
	datatype, ok := scalars[f.Type.Name]
	if !ok {
		return nil, fmt.Errorf("unknown type %q", f.Type.Name)
	}
	node.Datatype = datatype
	if f.Type.List {
		node.Datatype += "[]"
	}
	return node, nil
}

// upperFirst renders a GraphQL field name the way VSS writes a path segment.
//
// VSS paths are PascalCase -- `Vehicle.Cabin.Seat.IsBelted` -- and a
// converted node must look the same, or a consumer joining on fqn would have
// to know which source a resource came from.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
