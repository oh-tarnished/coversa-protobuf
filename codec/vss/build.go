// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vss

// build.go extracts the manifest from the model.
//
// It reads the same model the schema is generated from rather than the
// emitted .proto files, for the reason the docs generator does: the mapping
// records decisions the generator *made* -- which field was renamed, which
// unit was pinned -- and recovering them from the output would mean deriving
// them a second time.

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/describe"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// Build extracts the manifest from a built model.
func Build(m *model.Model) *Manifest {
	out := &Manifest{
		Version:   m.Spec.Version,
		Commit:    m.Spec.Commit,
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
func resourceOf(m *model.Model, planner *plan.Planner, pkg *model.Package, t *sdl.Def) *Resource {
	r := &Resource{Fields: map[string]*Field{}}
	if v, ok := t.Directive("vspec"); ok {
		r.FQN, _ = v.Arg("fqn")
	}
	if t.Name == pkg.Root.Name {
		r.Pattern = pkg.Pattern
	}

	isRoot := t.Name == pkg.Root.Name
	for _, f := range t.Fields {
		// A field naming a resource in another package became a resource
		// name, which carries no VSS signal of its own.
		if _, isObject := m.Types[f.Type.Name]; isObject && !pkg.Holds(f.Type.Name) {
			continue
		}
		p, err := planner.Field(t, f, isRoot)
		if err != nil {
			continue
		}
		mapped := fieldOf(m, f, p)

		// The instance-tag axes carry no @vspec, because VSS does not model
		// an instance as a signal -- it expands the path, so a seat's height
		// is `Vehicle.Cabin.Seat.Row1.DriverSide.Height`.
		//
		// They are still the only thing saying *which* seat a reading is
		// from, so dropping them would make a converted message ambiguous.
		// Each gets an address derived from the branch it identifies, marked
		// as an axis rather than claimed to be a signal VSS declares.
		if mapped.FQN == "" && r.FQN != "" {
			mapped.FQN = r.FQN + "." + naming.Pascal(p.Name)
			mapped.Element = "instance_axis"
		}
		r.Fields[p.Name] = mapped
	}
	return r
}

// fieldOf maps one field.
func fieldOf(m *model.Model, src sdl.Field, p plan.Field) *Field {
	out := &Field{
		FQN:      p.FQN,
		Element:  strings.ToLower(strings.TrimPrefix(p.Element, "ELEMENT_")),
		Quantity: strings.ToLower(strings.TrimPrefix(p.Quantity, "QUANTITY_KIND_")),
	}
	if p.Unit != "" {
		out.Unit = describe.UnitSymbol(strings.TrimPrefix(p.Unit, "UNIT_"))
	}
	// Recorded only where the two differ: a mapping that restates the field's
	// own name for 800 fields buries the ~30 that were actually renamed.
	if source := naming.Snake(src.Name); source != p.Name {
		out.Source = src.Name
	}
	if e, ok := m.Enums[src.Type.Name]; ok && !catalog.IsScalarEnum(e.Name) && !model.IsUnitEnum(e.Name) {
		out.Enum = m.EnumName(e.Name)
		out.Values = enumValues(m, e)
	}
	return out
}

// enumValues maps each emitted constant to the spelling a VSS-native peer
// sends.
func enumValues(m *model.Model, e *sdl.Def) map[string]string {
	prefix := naming.Screaming(m.EnumName(e.Name))

	out := make(map[string]string, len(e.Values))
	for i, v := range e.Values {
		source := v.Name
		if s, ok := describe.SourceSpelling(v); ok {
			source = s
		}
		out[plan.EnumValueName(prefix, v, i)] = source
	}
	return out
}
