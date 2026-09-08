// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// types.go maps one SDL field onto one protobuf field: its type, its name, the
// behaviour and validation annotations it carries, and the VSS annotation that
// records what protobuf cannot say.

import (
	"strconv"
	"strings"
)

// isNumericProto reports whether a protobuf type accepts a numeric range rule.
func isNumericProto(t string) bool {
	switch t {
	case "int32", "int64", "double", "float":
		return true
	}
	return false
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

// enumName renders a source enum name as a protobuf enum name. The `Vehicle_`
// prefix every allowed-value enum carries is redundant inside a package that
// is already part of the vehicle, and the `_Enum` suffix says nothing.
func enumName(g string) string {
	g = strings.TrimPrefix(g, "Vehicle_")
	g = strings.TrimSuffix(g, "_Enum")
	n := pascal(g)
	// AIP-216 prefers "State" over "Status" for a lifecycle state enum, and
	// reads any enum ending in Status as one. VSS uses Status for a physical
	// position -- a roof, a massage programme, a seat's occupancy -- so the
	// suffix is renamed and the field that names the enum keeps VSS's own
	// term through fieldRenames.
	if strings.HasSuffix(n, "Status") {
		n = strings.TrimSuffix(n, "Status") + "State"
	}
	return n
}

// tighter returns whichever of two bounds constrains more. `low` selects the
// direction: for a lower bound the larger value is tighter, for an upper bound
// the smaller one is.
func tighter(have, want string, low bool) string {
	if have == "" {
		return want
	}
	h, err1 := strconv.ParseFloat(have, 64)
	w, err2 := strconv.ParseFloat(want, 64)
	if err1 != nil || err2 != nil {
		return want
	}
	if low == (w > h) {
		return want
	}
	return have
}

// numLit renders a bound as a literal the field's protobuf type accepts. An
// integer rule will not take `0.0`, and a double rule will not take a bare
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

// messageName renders a source object type as a protobuf message name.
func messageName(g string) string {
	if r, ok := typeRenames[g]; ok {
		return r
	}
	return pascal(g)
}

// containsWord reports whether a split identifier contains the given word.
func containsWord(words []string, w string) bool {
	for _, x := range words {
		if x == w {
			return true
		}
	}
	return false
}
