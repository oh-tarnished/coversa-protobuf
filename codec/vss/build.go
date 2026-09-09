// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package vss

// build.go extracts the manifest from the model.
//
// It reads the same model the schema is generated from rather than the
// emitted .proto files, for the reason the docs generator does: the mapping
// records decisions the generator *made* -- which field was renamed, which
// unit was pinned -- and recovering them from the output would mean deriving
// them a second time, from text, and getting a different answer the first
// time the two drifted.

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// Build extracts the manifest from a built model.
func Build(m *model.Model) *Manifest {
	out := &Manifest{
		Spec: Spec{
			VSS: Revision{Version: m.Spec.VSS.Version, Commit: m.Spec.VSS.Commit},
			VDM: Revision{Version: m.Spec.VDM.Version, Commit: m.Spec.VDM.Commit},
		},
		Resources: map[string]*Resource{},
	}
	planner := plan.New(m)

	for _, pkg := range m.Packages {
		for _, t := range pkg.Types {
			full := pkg.ProtoPackage() + "." + model.MessageName(t.Name)
			out.Resources[full] = resourceOf(m, planner, pkg, t)
		}
	}
	return out
}

// resourceOf maps one message.
//
// The package root is also a resource, so it carries the name pattern and the
// instance axes a consumer needs to take an id apart. An embedded message
// carries neither: it is addressed as part of its owner.
func resourceOf(m *model.Model, planner *plan.Planner, pkg *model.Package, t *vspec.Node) *Resource {
	r := &Resource{FQN: t.FQN, Fields: map[string]*Field{}}

	isRoot := t == pkg.Root
	if isRoot {
		r.Pattern = pkg.Pattern
		r.Instances = t.Instances
	}

	for _, child := range t.Children {
		// A branch promoted to a resource of its own is not a field here: it
		// is reachable by a name derived from this one, and the emitter drops
		// it for the same reason. Mirrored rather than re-derived -- a
		// manifest naming a field the schema does not emit is worse than no
		// manifest, because a converter would trust it.
		if child.Kind == vspec.KindBranch && child.Ref == "" && !pkg.Holds(child.FQN) {
			continue
		}
		p, err := planner.Field(t, child, isRoot)
		if err != nil {
			continue
		}
		r.Fields[p.Name] = fieldOf(m, pkg, child, p)
	}
	return r
}

// fieldOf maps one field.
func fieldOf(m *model.Model, pkg *model.Package, n *vspec.Node, p plan.Field) *Field {
	out := &Field{
		FQN:     p.FQN,
		Element: strings.ToLower(strings.TrimPrefix(p.Element, "ELEMENT_")),
		Unit:    n.Unit,
	}
	if u, ok := m.Units.Units[n.Unit]; ok {
		out.Quantity = u.Quantity
	}

	// Recorded only where the two differ: a mapping that restates the field's
	// own name for 800 fields buries the ~30 that were actually renamed.
	if source := naming.Snake(n.Name); source != p.Name {
		out.Source = n.Name
	}

	switch {
	case n.Kind == vspec.KindBranch && n.Ref == "":
		// VSS models a branch as a node rather than as a signal, so there is
		// no element kind to report and the schema supplies its own.
		out.Element = "branch"
		out.Message = pkg.ProtoPackage() + "." + model.MessageName(n.Name)
	case model.HasEnum(n):
		out.Enum = m.EnumName(n.FQN)
		out.Values = enumValues(m, n)
	}
	return out
}

// enumValues maps each emitted constant to the spelling a VSS-native peer
// sends.
//
// Both halves come from the same two functions the emitter uses. Spelling the
// constants a second time here is how the manifest would come to name values
// no generated enum has.
func enumValues(m *model.Model, n *vspec.Node) map[string]string {
	prefix := naming.Screaming(m.EnumName(n.FQN))
	values := plan.EnumValues(n)

	out := make(map[string]string, len(values))
	for _, v := range values {
		out[plan.EnumValueName(prefix, v.Name)] = v.Name
	}
	return out
}
