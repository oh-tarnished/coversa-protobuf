// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package plan

// types.go resolves the protobuf type for a field, and the imports that type
// pulls in.

import (
	"fmt"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// assignType resolves the protobuf type for a field.
func (p *Planner) assignType(owner *sdl.Def, f sdl.Field, unitEnum string, out *Field) error {
	if catalog.DurationFields[owner.Name+"."+f.Name] {
		out.Type = "google.protobuf.Duration"
		out.Imports = append(out.Imports, "google/protobuf/duration.proto")
		out.Unit, out.Quantity = "", ""
		return nil
	}
	if name := f.Type.Name; name == "String" && temporal(f, unitEnum) {
		p.assignTemporal(f, out)
		return nil
	}
	if w, ok := catalog.IntegerWidths[f.Type.Name]; ok {
		out.Type = w.Proto
		out.NumLo, out.NumHi = w.Lo, w.Hi
		return nil
	}
	if t, ok := scalars[f.Type.Name]; ok {
		out.Type = t
		return nil
	}
	return p.assignNamed(owner, f, out)
}

// scalars maps the GraphQL built-in types onto protobuf ones.
//
// Float widens to double: GraphQL's Float is an IEEE 754 double and the
// specification says nothing narrower, so halving the wire size would discard
// precision the source does not say is absent. See docs/decisions.md.
var scalars = map[string]string{
	"String":  "string",
	"ID":      "string",
	"Boolean": "bool",
	"Float":   "double",
	"Int":     "int32",
}

// temporal reports whether a String field actually holds a time or a date.
//
// VSS tags most of them with a DatetimeUnitEnum argument; the VDM types
// outside the vehicle tree state an ISO 8601 string with no argument at all,
// and AIP-142 reads both the same way -- by the name.
func temporal(f sdl.Field, unitEnum string) bool {
	if unitEnum == "DatetimeUnitEnum" {
		return true
	}
	words := naming.SplitWords(f.Name)
	return naming.ContainsWord(words, "time") || naming.ContainsWord(words, "date")
}

// assignTemporal types a temporal string as a Date or a Timestamp.
//
// Typing it is the whole reason AIP-142 cares what such a field is called: a
// date-only value has no time zone, an instant does, and a string cannot say
// which it is.
func (p *Planner) assignTemporal(f sdl.Field, out *Field) {
	if naming.ContainsWord(naming.SplitWords(f.Name), "date") {
		out.Type = "Date"
	} else {
		out.Type = "google.protobuf.Timestamp"
		out.Imports = append(out.Imports, "google/protobuf/timestamp.proto")
	}
	// The unit said "an ISO 8601 string"; the protobuf type now says it
	// better, so the annotation would only disagree with the field.
	out.Unit, out.Quantity = "", ""
}

// assignNamed resolves a reference to an enum or an object type.
func (p *Planner) assignNamed(owner *sdl.Def, f sdl.Field, out *Field) error {
	name := f.Type.Name

	if se, ok := catalog.ScalarEnums[name]; ok {
		out.Type = se.Proto
		out.Validate = append(out.Validate, se.Rule,
			"(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
		if out.Doc != "" {
			out.Doc += "\n\n"
		}
		out.Doc += se.Doc
		return nil
	}
	if e, ok := p.M.Enums[name]; ok {
		out.Type = p.M.EnumName(e.Name)
		out.Validate = append(out.Validate, "(buf.validate.field).enum.defined_only = true")
		return nil
	}
	if t, ok := p.M.Types[name]; ok {
		out.Type = model.MessageName(t.Name)
		return nil
	}
	return fmt.Errorf("%s.%s: unknown type %q", owner.Name, f.Name, name)
}

// normaliseUnit maps a source unit spelling onto this schema's Unit enum.
func normaliseUnit(v string) string {
	if v == "U_N_I_X_TIMESTAMP" {
		// A sanitisation artefact upstream: VSS writes "UNIX Timestamp", and
		// splitting it on case produced one letter per word.
		return "UNIX_TIMESTAMP"
	}
	return v
}
