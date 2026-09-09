// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package catalog

// units.go renders VSS unit and quantity names as protobuf enum constants.
//
// The names themselves are not listed here. VSS ships units.yaml and
// quantities.yaml, package vspec reads them, and this file only converts a
// symbol into the constant that names it. A transcribed table would be a
// second copy of a published vocabulary, and would drift the first time VSS
// added a unit.

import "strings"

// symbolWords spells out the characters a unit symbol uses that cannot appear
// in an identifier.
//
// A symbol is written for a person -- `km/h`, `m/s^2`, `l/100km` -- and an
// enum constant cannot hold a slash or a caret. Spelling them keeps the
// constant readable and, more importantly, keeps two different symbols from
// colliding: dropping the punctuation would make `m/s` and `ms` the same.
var symbolWords = strings.NewReplacer(
	"/", "_PER_",
	"^", "_POW_",
	".", "_",
	"-", "_",
	" ", "_",
	"%", "PERCENT",
)

// UnitConstant renders a VSS unit symbol as its Unit enum constant.
func UnitConstant(symbol string) string {
	if symbol == "" {
		return ""
	}
	return "UNIT_" + identifier(symbol)
}

// QuantityConstant renders a VSS quantity name as its QuantityKind constant.
func QuantityConstant(quantity string) string {
	if quantity == "" {
		return ""
	}
	return "QUANTITY_KIND_" + identifier(quantity)
}

// identifier turns a symbol or name into the body of an enum constant.
func identifier(s string) string {
	out := symbolWords.Replace(s)

	var b strings.Builder
	for _, r := range out {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return collapse(b.String())
}

// collapse squeezes runs of underscores and trims the ends, so `l/100km`
// does not become `L_PER_100KM_`.
func collapse(s string) string {
	var b strings.Builder
	prev := '_'
	for _, r := range s {
		if r == '_' && prev == '_' {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return strings.Trim(b.String(), "_")
}
