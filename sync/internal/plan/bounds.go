// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package plan

// bounds.go turns numeric constraints into buf.validate rules, and renders
// the enum names those rules refer to.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/internal/naming"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// bounds intersects the width bound with any @range and emits one rule.
//
// A signal may be bounded twice: once by the width VSS declares -- a UInt8
// cannot exceed 255 -- and once by an explicit @range. Protobuf accepts a
// single rule per field and emitting two is a compile error, so the two are
// intersected and the tighter bound on each side wins.
func (p *Planner) bounds(f sdl.Field, out *Field) {
	if r, ok := f.Directive("range"); ok {
		if lo, has := r.Arg("min"); has {
			out.NumLo = tighter(out.NumLo, lo, true)
		}
		if hi, has := r.Arg("max"); has {
			out.NumHi = tighter(out.NumHi, hi, false)
		}
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

// EnumName renders a source enum name as a protobuf enum name.
//
// The `Vehicle_` prefix every allowed-value enum carries is redundant inside a
// package that is already part of the vehicle, and the `_Enum` suffix says
// nothing. AIP-216 then prefers "State" over "Status", and reads any enum
// ending in Status as a lifecycle enum -- VSS uses Status for a physical
// position, so the suffix is renamed and the field keeps VSS's own term
// through the catalogue.
func EnumName(g string) string {
	g = strings.TrimPrefix(g, "Vehicle_")
	g = strings.TrimSuffix(g, "_Enum")
	n := naming.Pascal(g)
	if strings.HasSuffix(n, "Status") {
		n = strings.TrimSuffix(n, "Status") + "State"
	}
	return n
}
