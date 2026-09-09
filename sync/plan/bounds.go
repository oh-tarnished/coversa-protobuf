// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package plan

// bounds.go turns numeric constraints into buf.validate rules, and renders
// the enum names those rules refer to.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// bounds intersects the width bound with any @range and emits one rule.
//
// A signal may be bounded twice: once by the width VSS declares -- a UInt8
// cannot exceed 255 -- and once by an explicit @range. Protobuf accepts a
// single rule per field and emitting two is a compile error, so the two are
// intersected and the tighter bound on each side wins.
func (p *Planner) bounds(n *vspec.Node, out *Field) {
	if n.Min != "" {
		out.NumLo = tighter(out.NumLo, n.Min, true)
	}
	if n.Max != "" {
		out.NumHi = tighter(out.NumHi, n.Max, false)
	}
	if !numericProto(out.Type) || (out.NumLo == "" && out.NumHi == "") {
		return
	}

	var parts []string
	if out.NumLo != "" {
		parts = append(parts, "gte: "+numLit(out.NumLo, out.Type))
	}
	if out.NumHi != "" {
		parts = append(parts, "lte: "+numLit(out.NumHi, out.Type))
	}
	out.Validate = append(out.Validate, rangeRule(out.Type, parts))
}

// rangeRule renders a numeric rule the way buf format would write it.
//
// buf collapses a one-field message literal onto its own line and expands
// anything larger. Matching that here keeps the generator's raw output
// identical to the formatted tree, so CI's regenerate-and-diff compares
// exactly rather than approximately.
func rangeRule(protoType string, parts []string) string {
	if len(parts) == 1 {
		return fmt.Sprintf("(buf.validate.field).%s = {%s}", protoType, parts[0])
	}
	return fmt.Sprintf("(buf.validate.field).%s = {\n      %s\n    }",
		protoType, strings.Join(parts, "\n      "))
}

// numericProto reports whether a protobuf type accepts a numeric range rule.
func numericProto(t string) bool {
	switch t {
	case "int32", "int64", "double", "float":
		return true
	}
	return false
}

// tighter returns whichever of two bounds constrains more.
//
// low selects the direction: for a lower bound the larger value is tighter,
// for an upper bound the smaller one is.
func tighter(have, want string, low bool) string {
	if have == "" {
		return want
	}
	h, errHave := strconv.ParseFloat(have, 64)
	w, errWant := strconv.ParseFloat(want, 64)
	if errHave != nil || errWant != nil {
		return want
	}
	if low == (w > h) {
		return want
	}
	return have
}

// numLit renders a bound as a literal the field's protobuf type accepts.
//
// An integer rule will not take `0.0`, and a double rule will not take a bare
// integer in every protobuf version, so each is spelled for its own type.
func numLit(v, protoType string) string {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	if protoType == "int32" || protoType == "int64" {
		return strconv.FormatInt(int64(f), 10)
	}
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10) + ".0"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// EnumValueName renders one enumerant as the constant the schema emits.
//
// AIP-126 prefixes every value with its enum's name, because proto3 scopes
// enumerants in the *package* rather than the enum. The suffix keeps the
// result clear of the keywords a target reserves once it strips that prefix
// back off again -- see catalog.EnumValueSuffix.
func EnumValueName(prefix, value string) string {
	return prefix + "_" + naming.Screaming(value) + catalog.EnumValueSuffix(value)
}

// EnumValues returns a node's allowed values as name/number pairs.
//
// VSS states them two ways. `enum` carries the specification's own numbering
// and is used as written; `allowed` is a bare list, which is numbered from
// one so that zero stays free for the "unspecified" AIP-126 requires.
func EnumValues(n *vspec.Node) []vspec.EnumEntry {
	if len(n.Enum) > 0 {
		return n.Enum
	}
	out := make([]vspec.EnumEntry, 0, len(n.Allowed))
	for i, name := range n.Allowed {
		out = append(out, vspec.EnumEntry{Name: name, Number: i + 1})
	}
	return out
}
