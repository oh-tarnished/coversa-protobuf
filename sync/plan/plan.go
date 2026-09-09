// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

// Package plan maps one specification node onto one protobuf field: its name,
// its type, the behaviour and validation annotations it carries, and the VSS
// annotation that records what protobuf cannot say.
//
// The output is a [Field] — a description of the field, not its text. Package
// emit turns it into protobuf syntax. Keeping the two apart means the
// decisions here can be tested without matching whitespace.
package plan

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/catalog"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// Field is one planned protobuf field, before it is written out.
type Field struct {
	Name     string
	Type     string
	Repeated bool
	Doc      string
	Comment  string

	// Note is documentation the schema owes the reader about a decision it
	// made — why a country code is a string rather than an enum. The
	// specification does not say it, because the specification did not make
	// the decision.
	Note string

	// Behavior holds the google.api.field_behavior values, and Validate the
	// rendered buf.validate rules.
	Behavior []string
	Validate []string

	// NumLo and NumHi are the numeric bounds before rendering. Held
	// separately so a width bound and a declared range can be intersected
	// rather than both emitted.
	NumLo string
	NumHi string

	// Element, FQN, Unit and Quantity become the VSS signal annotation.
	Element  string
	FQN      string
	Unit     string
	Quantity string

	// Ref names the resource this field points at, where it is an
	// association rather than a value.
	Ref string

	// Deprecated is the reason, where the specification deprecates the signal.
	Deprecated string

	// Imports are the well-known proto files this field's type pulls in.
	Imports []string
}

// Planner maps nodes against an indexed model.
type Planner struct{ M *model.Model }

// New returns a Planner over m.
func New(m *model.Model) *Planner { return &Planner{M: m} }

// Field maps one specification node onto a protobuf field.
//
// isRoot marks a field on a package's resource message, where the AIP
// identity and lifecycle fields already occupy several names.
func (p *Planner) Field(owner, n *vspec.Node, isRoot bool) (Field, error) {
	out := Field{
		Doc:        n.Description,
		Comment:    n.Comment,
		Repeated:   n.Repeated || repeatedDatatype(n.Datatype),
		FQN:        n.FQN,
		Ref:        n.Ref,
		Deprecated: n.Deprecation,
	}
	if n.Kind.Signal() {
		out.Element = "ELEMENT_" + strings.ToUpper(string(n.Kind))
	}
	if err := p.name(owner, n, isRoot, &out); err != nil {
		return out, err
	}
	if err := p.assignType(owner, n, &out); err != nil {
		return out, err
	}
	p.unit(n, &out)
	p.behavior(n, &out)
	p.bounds(n, &out)
	return out, nil
}

// name resolves the protobuf field name, applying the catalogue and the
// reserved-word rule, and rejecting a collision with a resource's own fields.
func (p *Planner) name(owner, n *vspec.Node, isRoot bool, out *Field) error {
	out.Name = naming.Snake(n.Name)
	if r, ok := catalog.FieldRenames[n.FQN]; ok {
		out.Name = r
	}
	// AIP-140 forbids a field named for a keyword in a common target
	// language. Applied by rule, so a new one in a later release is caught
	// rather than shipped.
	if catalog.ReservedWord(out.Name) {
		out.Name += "_control"
	}
	if isRoot && catalog.ReservedResourceFields[out.Name] {
		return fmt.Errorf("%s maps to %q, which a resource's own AIP fields "+
			"already occupy; add a rename to FieldRenames in internal/catalog",
			n.FQN, out.Name)
	}
	return nil
}

// unit records the unit the specification pins for this signal.
//
// VSS states one unit per signal, so this is a fact about what the number
// means rather than a choice the message carries. Protobuf has nowhere to put
// it, which is why the annotation vocabulary exists.
func (p *Planner) unit(n *vspec.Node, out *Field) {
	if n.Unit == "" {
		return
	}
	out.Unit = catalog.UnitConstant(n.Unit)

	// The quantity comes from the catalogue, not from the symbol: `km/h`
	// measures velocity, and nothing in the three characters says so.
	if u, ok := p.M.Units.Units[n.Unit]; ok {
		out.Quantity = catalog.QuantityConstant(u.Quantity)
	}
}

// behavior sets the field_behavior a signal's VSS kind implies.
//
// A sensor is measured and an attribute is fixed at build time. Neither is
// writable through this API, which is what OUTPUT_ONLY says; an actuator is
// the only writable kind.
func (p *Planner) behavior(n *vspec.Node, out *Field) {
	if n.Ref != "" {
		out.Behavior = []string{"OPTIONAL"}
		if n.Required {
			// A required association to a past event does not change after
			// the event.
			out.Behavior = []string{"REQUIRED", "IMMUTABLE"}
		}
		return
	}
	switch n.Kind {
	case vspec.KindSensor, vspec.KindAttribute:
		out.Behavior = []string{"OUTPUT_ONLY"}
	default:
		out.Behavior = []string{"OPTIONAL"}
	}
}
