// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// plan.go maps one SDL field onto one protobuf field: its name, its type, the
// behaviour and validation annotations it carries, and the VSS annotation
// that records what protobuf cannot say.
import (
	"fmt"
	"strings"
)

// FieldPlan is one rendered protobuf field, before it is written out.
type FieldPlan struct {
	Name       string
	Type       string
	Repeated   bool
	Doc        string
	Behavior   []string
	Validate   []string
	NumLo      string
	NumHi      string
	Element    string
	FQN        string
	Unit       string
	Quantity   string
	Deprecated string
	Imports    []string
}

// planField maps one SDL field onto a protobuf field.
func (m *Model) planField(owner *Def, f Field, isRoot bool) (FieldPlan, error) {
	p := FieldPlan{Doc: f.Doc, Repeated: f.Type.List}

	p.Name = snake(f.Name)
	if r, ok := fieldRenames[owner.Name+"."+f.Name]; ok {
		p.Name = r
	}
	// AIP-140 forbids a field named for a keyword in a common target
	// language. Applied by rule rather than by a list of the eight `switch`
	// fields VSS has today, so a new one in a later release is caught rather
	// than shipped.
	if reservedWords[p.Name] {
		p.Name += "_control"
	}
	if isRoot && reservedResourceFields[p.Name] {
		return p, fmt.Errorf("%s.%s maps to %q, which a resource's own AIP fields "+
			"already occupy; add a rename to fieldRenames in tools/vssgen/types.go",
			owner.Name, f.Name, p.Name)
	}

	if v, ok := f.Directive("vspec"); ok {
		if e, ok := v.Arg("element"); ok {
			p.Element = "ELEMENT_" + e
		}
		if q, ok := v.Arg("fqn"); ok {
			p.FQN = q
		}
	}
	if d, ok := f.Directive("deprecated"); ok {
		p.Deprecated, _ = d.Arg("reason")
		if p.Deprecated == "" {
			p.Deprecated = "deprecated in the source model"
		}
	}

	// The unit lives on a field argument -- `speed(unit: VelocityUnitEnum =
	// KILOMETER_PER_HOUR)` -- with the canonical unit as the default. Protobuf
	// has no field arguments, so the default is pinned and recorded.
	var unitEnum string
	for _, a := range f.Args {
		if a.Name == "unit" && isUnitEnum(a.Type.Name) {
			unitEnum = a.Type.Name
			if a.Default != "" {
				p.Unit = "UNIT_" + normaliseUnit(a.Default)
				p.Quantity = "QUANTITY_KIND_" + screaming(strings.TrimSuffix(a.Type.Name, unitEnumSuffix))
			}
		}
	}

	if err := m.assignType(owner, f, unitEnum, &p); err != nil {
		return p, err
	}

	if roundTripIDs[owner.Name+"."+f.Name] {
		// Rule 9: the originating system's identifier, not ours. It is
		// whatever that system chose and is preserved exactly as supplied.
		p.Behavior = []string{"OPTIONAL", "IMMUTABLE"}
		p.Doc += "\n\nThe identifier the originating system assigned, distinct " +
			"from the server-assigned `uid` above. It arrives inside imported data " +
			"and must survive a round trip unchanged, or every re-import duplicates " +
			"the record."
		return p, nil
	}

	switch p.Element {
	case "ELEMENT_SENSOR", "ELEMENT_ATTRIBUTE":
		// A sensor is measured and an attribute is fixed at build time.
		// Neither is writable through this API, which is what OUTPUT_ONLY
		// says; an actuator below is the only writable kind.
		p.Behavior = []string{"OUTPUT_ONLY"}
	default:
		p.Behavior = []string{"OPTIONAL"}
	}

	// A signal may be bounded twice: once by the width VSS declares -- a
	// UInt8 cannot exceed 255 -- and once by an explicit @range. Protobuf
	// accepts one rule per field, so the two are intersected rather than both
	// emitted, and the tighter bound on each side wins.
	if r, ok := f.Directive("range"); ok {
		if lo, has := r.Arg("min"); has {
			p.NumLo = tighter(p.NumLo, lo, true)
		}
		if hi, has := r.Arg("max"); has {
			p.NumHi = tighter(p.NumHi, hi, false)
		}
	}
	if isNumericProto(p.Type) && (p.NumLo != "" || p.NumHi != "") {
		var parts []string
		if p.NumLo != "" {
			parts = append(parts, "gte: "+numLit(p.NumLo, p.Type))
		}
		if p.NumHi != "" {
			parts = append(parts, "lte: "+numLit(p.NumHi, p.Type))
		}
		// buf format collapses a one-field message literal onto its own line
		// and expands anything larger. Matching that here keeps the
		// generator's raw output identical to the formatted tree, so CI's
		// regenerate-and-diff compares exactly rather than approximately.
		if len(parts) == 1 {
			p.Validate = append(p.Validate,
				fmt.Sprintf("(buf.validate.field).%s = {%s}", p.Type, parts[0]))
		} else {
			p.Validate = append(p.Validate,
				fmt.Sprintf("(buf.validate.field).%s = {\n      %s\n    }",
					p.Type, strings.Join(parts, "\n      ")))
		}
	}
	return p, nil
}

// assignType resolves the protobuf type for a field and records the imports it
// pulls in.
func (m *Model) assignType(owner *Def, f Field, unitEnum string, p *FieldPlan) error {
	name := f.Type.Name

	if durationFields[owner.Name+"."+f.Name] {
		p.Type = "google.protobuf.Duration"
		p.Imports = append(p.Imports, "google/protobuf/duration.proto")
		p.Unit, p.Quantity = "", ""
		return nil
	}

	// An ISO 8601 string is a time, and typing it as one is the whole reason
	// AIP-142 cares what a field is called. A date-only value becomes
	// google.type.Date, an instant becomes google.protobuf.Timestamp; both are
	// `google.*`, the one package AIP-215 lets a schema share.
	// A string that names a time is a time. VSS tags most of them with a
	// DatetimeUnitEnum argument; the VDM types outside the vehicle tree state
	// an ISO 8601 string with no argument at all, and AIP-142 reads both the
	// same way -- by the name.
	words := splitWords(f.Name)
	isTemporal := unitEnum == "DatetimeUnitEnum" ||
		containsWord(words, "time") || containsWord(words, "date")
	if name == "String" && isTemporal {
		if containsWord(words, "date") {
			p.Type = "Date"
		} else {
			p.Type = "google.protobuf.Timestamp"
			p.Imports = append(p.Imports, "google/protobuf/timestamp.proto")
		}
		// The unit said "an ISO 8601 string"; the protobuf type now says it
		// better, so the annotation would only disagree with the field.
		p.Unit, p.Quantity = "", ""
		return nil
	}

	if w, ok := integerWidths[name]; ok {
		p.Type = w.proto
		p.NumLo, p.NumHi = w.lo, w.hi
		return nil
	}

	switch name {
	case "String", "ID":
		p.Type = "string"
	case "Boolean":
		p.Type = "bool"
	case "Float":
		// GraphQL Float is an IEEE 754 double and the spec says nothing
		// narrower, so widening here is the only lossless choice. See
		// docs/decisions.md for why the halved wire size was not taken.
		p.Type = "double"
	case "Int":
		p.Type = "int32"
	default:
		if se, ok := scalarEnums[name]; ok {
			p.Type = se.proto
			p.Validate = append(p.Validate, se.rule,
				"(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
			if p.Doc != "" {
				p.Doc += "\n\n"
			}
			p.Doc += se.doc
			return nil
		}
		if e, ok := m.Enums[name]; ok {
			p.Type = enumName(e.Name)
			p.Validate = append(p.Validate, "(buf.validate.field).enum.defined_only = true")
			return nil
		}
		if t, ok := m.Types[name]; ok {
			p.Type = messageName(t.Name)
			return nil
		}
		return fmt.Errorf("%s.%s: unknown type %q", owner.Name, f.Name, name)
	}
	return nil
}
