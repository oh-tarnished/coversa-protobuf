// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package plan maps one SDL field onto one protobuf field: its name, its
// type, the behaviour and validation annotations it carries, and the VSS
// annotation that records what protobuf cannot say.
//
// The output is a [Field] -- a description of the field, not its text. The
// emit package turns it into protobuf syntax. Keeping the two apart means the
// decisions here can be tested without matching whitespace.
package plan

import (
	"fmt"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// Field is one planned protobuf field, before it is written out.
type Field struct {
	Name     string
	Type     string
	Repeated bool
	Doc      string

	// Behavior holds the google.api.field_behavior values, and Validate the
	// rendered buf.validate rules.
	Behavior []string
	Validate []string

	// NumLo and NumHi are the numeric bounds, before they are rendered into
	// Validate. Held separately so a width bound and a @range bound can be
	// intersected rather than both emitted.
	NumLo string
	NumHi string

	// Element, FQN, Unit and Quantity become the VSS signal annotation.
	Element  string
	FQN      string
	Unit     string
	Quantity string

	// Deprecated is the reason, when the source model deprecates the signal.
	Deprecated string

	// Imports are the well-known proto files this field's type pulls in.
	Imports []string
}

// Planner maps fields against an indexed model.
type Planner struct{ M *model.Model }

// New returns a Planner over m.
func New(m *model.Model) *Planner { return &Planner{M: m} }

// Field maps one SDL field onto a protobuf field.
//
// isRoot marks a field on a package's resource message, where the AIP
// identity and lifecycle fields already occupy several names.
func (p *Planner) Field(owner *sdl.Def, f sdl.Field, isRoot bool) (Field, error) {
	out := Field{Doc: f.Doc, Repeated: f.Type.List}
	if err := p.name(owner, f, isRoot, &out); err != nil {
		return out, err
	}
	p.annotations(f, &out)

	unitEnum := p.unit(f, &out)
	if err := p.assignType(owner, f, unitEnum, &out); err != nil {
		return out, err
	}

	if catalog.RoundTripIDs[owner.Name+"."+f.Name] {
		p.roundTripID(&out)
		return out, nil
	}
	p.behavior(&out)
	p.bounds(f, &out)
	return out, nil
}

// name resolves the protobuf field name, applying the catalogue and the
// reserved-word rule, and rejecting a collision with a resource's own fields.
func (p *Planner) name(owner *sdl.Def, f sdl.Field, isRoot bool, out *Field) error {
	out.Name = naming.Snake(f.Name)
	if r, ok := catalog.FieldRenames[owner.Name+"."+f.Name]; ok {
		out.Name = r
	}
	// AIP-140 forbids a field named for a keyword in a common target
	// language. Applied by rule, so a new one in a later release is caught
	// rather than shipped.
	if catalog.ReservedWord(out.Name) {
		out.Name += "_control"
	}
	if isRoot && catalog.ReservedResourceFields[out.Name] {
		return fmt.Errorf("%s.%s maps to %q, which a resource's own AIP fields "+
			"already occupy; add a rename to FieldRenames in internal/catalog",
			owner.Name, f.Name, out.Name)
	}
	return nil
}

// annotations reads the VSS element kind, fully qualified name and any
// deprecation reason off the field's directives.
func (p *Planner) annotations(f sdl.Field, out *Field) {
	if v, ok := f.Directive("vspec"); ok {
		if e, ok := v.Arg("element"); ok {
			out.Element = "ELEMENT_" + e
		}
		if q, ok := v.Arg("fqn"); ok {
			out.FQN = q
		}
	}
	if d, ok := f.Directive("deprecated"); ok {
		out.Deprecated, _ = d.Arg("reason")
		if out.Deprecated == "" {
			out.Deprecated = "deprecated in the source model"
		}
	}
}

// unit pins the signal's unit and returns the unit enum it came from.
//
// The unit lives on a field argument -- `speed(unit: VelocityUnitEnum =
// KILOMETER_PER_HOUR)` -- with the canonical unit as the default. Protobuf
// has no field arguments, so the default is pinned and recorded here.
func (p *Planner) unit(f sdl.Field, out *Field) string {
	for _, a := range f.Args {
		if a.Name != "unit" || !model.IsUnitEnum(a.Type.Name) {
			continue
		}
		if a.Default != "" {
			out.Unit = "UNIT_" + normaliseUnit(a.Default)
			out.Quantity = "QUANTITY_KIND_" +
				naming.Screaming(strings.TrimSuffix(a.Type.Name, "UnitEnum"))
		}
		return a.Type.Name
	}
	return ""
}

// roundTripID marks the originating system's own identifier.
//
// Two identifiers, not one: `uid` is server-assigned and ours, this one
// arrives inside imported data and is whatever the originating system chose.
// It must survive a round trip unchanged or every re-import duplicates the
// record, which is why it is IMMUTABLE rather than merely optional.
func (p *Planner) roundTripID(out *Field) {
	out.Behavior = []string{"OPTIONAL", "IMMUTABLE"}

	const note = "The identifier the originating system assigned, distinct " +
		"from the server-assigned `uid` above. It arrives inside imported data " +
		"and must survive a round trip unchanged, or every re-import duplicates " +
		"the record."

	// Appended to the source description where there is one, and standing as
	// the description where there is not. Concatenating unconditionally left
	// the comment opening with two blank lines, and the Markdown reference --
	// which reads the first paragraph -- with nothing at all.
	if strings.TrimSpace(out.Doc) == "" {
		out.Doc = note
		return
	}
	out.Doc += "\n\n" + note
}

// behavior sets the field_behavior a signal's VSS kind implies.
//
// A sensor is measured and an attribute is fixed at build time. Neither is
// writable through this API, which is what OUTPUT_ONLY says; an actuator is
// the only writable kind.
func (p *Planner) behavior(out *Field) {
	switch out.Element {
	case "ELEMENT_SENSOR", "ELEMENT_ATTRIBUTE":
		out.Behavior = []string{"OUTPUT_ONLY"}
	default:
		out.Behavior = []string{"OPTIONAL"}
	}
}
