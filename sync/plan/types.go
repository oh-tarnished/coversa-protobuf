// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package plan

// types.go resolves the protobuf type for a field, and the imports it pulls
// in.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/catalog"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// assignType resolves the protobuf type for a node.
func (p *Planner) assignType(owner, n *vspec.Node, out *Field) error {
	switch {
	case n.Ref != "":
		// An association is the other resource's name, per AIP-122.
		out.Type = "string"
		return nil

	case n.Kind == vspec.KindBranch:
		// Through the same function the declaration uses: a message renamed
		// to satisfy an AIP rule -- Identifier becomes Identity -- must be
		// referenced by the new name, not the source one.
		out.Type = model.MessageName(n.Name)
		return nil

	case catalog.DurationFields[n.FQN]:
		out.Type = "google.protobuf.Duration"
		out.Imports = append(out.Imports, "google/protobuf/duration.proto")
		return nil

	case catalog.IsScalarEnum(n.EnumName):
		// An allowed-value set the catalogue replaces with a documented
		// scalar. The rule is what keeps the scalar honest once the enum's
		// exhaustiveness is gone, and the note is why it went.
		s := catalog.ScalarEnums[n.EnumName]
		out.Type = s.Proto
		out.Validate = append(out.Validate, s.Rule)
		out.Note = s.Doc
		return nil

	case model.HasEnum(n):
		out.Type = p.M.EnumName(n.FQN)
		out.Validate = append(out.Validate, "(buf.validate.field).enum.defined_only = true")
		return nil

	case temporal(n):
		p.assignTemporal(n, out)
		return nil
	}
	return p.assignScalar(n, out)
}

// datatypes maps VSS's scalar names onto protobuf types.
//
// AIP-141 forbids unsigned protobuf types, so every unsigned VSS width widens
// to a signed one; the bound the type just lost is restored as a
// buf.validate range, from Bounds below.
var datatypes = map[string]string{
	"boolean": "bool",
	"string":  "string",
	"float":   "double",
	"double":  "double",
	"int8":    "int32",
	"uint8":   "int32",
	"int16":   "int32",
	"uint16":  "int32",
	"int32":   "int32",
	"uint32":  "int64",
	"int64":   "int64",
	"uint64":  "int64",
}

// widthBounds is the range a VSS width admits, restored after AIP-141 widens
// the type past it.
var widthBounds = map[string][2]string{
	"int8": {"-128", "127"}, "uint8": {"0", "255"},
	"int16": {"-32768", "32767"}, "uint16": {"0", "65535"},
	"uint32": {"0", "4294967295"},
}

// assignScalar resolves a plain value type.
//
// `float` widens to `double`: VSS's float is single precision, but the
// specification's bounds and defaults are written in decimal and several do
// not survive a round trip through float32. See docs/decisions.md.
func (p *Planner) assignScalar(n *vspec.Node, out *Field) error {
	base := strings.TrimSuffix(n.Datatype, "[]")
	proto, ok := datatypes[base]
	if !ok {
		return fmt.Errorf("%s: unknown datatype %q", n.FQN, n.Datatype)
	}
	out.Type = proto
	if b, ok := widthBounds[base]; ok {
		out.NumLo, out.NumHi = b[0], b[1]
	}
	return nil
}

// repeatedDatatype reports whether a VSS datatype is an array.
func repeatedDatatype(datatype string) bool {
	return strings.HasSuffix(datatype, "[]")
}

// temporal reports whether a string field actually holds a time or a date.
//
// VSS marks these with the `iso8601` unit where it marks them at all; the
// rest are recognised by name, which is what AIP-142 reads too.
func temporal(n *vspec.Node) bool {
	if n.Datatype != "string" {
		return false
	}
	if strings.EqualFold(n.Unit, "iso8601") {
		return true
	}
	words := naming.SplitWords(n.Name)
	return naming.ContainsWord(words, "time") || naming.ContainsWord(words, "date")
}

// assignTemporal types a temporal string as a Date or a Timestamp.
//
// Typing it is the whole reason AIP-142 cares what such a field is called: a
// date-only value has no time zone, an instant does, and a string cannot say
// which it is.
func (p *Planner) assignTemporal(n *vspec.Node, out *Field) {
	if naming.ContainsWord(naming.SplitWords(n.Name), "date") {
		out.Type = "Date"
		return
	}
	out.Type = "google.protobuf.Timestamp"
	out.Imports = append(out.Imports, "google/protobuf/timestamp.proto")
}
