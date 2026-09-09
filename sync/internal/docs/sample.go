// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// sample.go invents the example values the reference shows.
//
// # Why they are derived rather than written
//
// Every page carries a worked example -- a resource as the API returns it,
// and the same resource in the VSS shape. Hand-written examples for 51
// packages would be 51 things to keep in step with a specification that
// moves, and the first one to drift would be indistinguishable from the rest.
//
// # Why they are deterministic
//
// CI regenerates the whole tree and diffs it. An example built from a random
// id or a clock would differ on every run, so the diff would report a change
// nobody made and the gate would be useless. Everything here is a pure
// function of the model: the same specification produces the same page,
// byte for byte.

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// crockford is Crockford base32: the digits, and the letters with I, L, O and
// U removed so nothing reads as a digit or as a slur.
//
// Lowercase because AIP-122 wants a resource id matching
// ^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$, and the alphabet is case-insensitive by
// specification, so lowercasing costs nothing.
const crockford = "0123456789abcdefghjkmnpqrstvwxyz"

// exampleVIN stands in for a vehicle's identifier throughout the reference.
//
// A real VIN shape: already unique, already external and meaningful, which is
// the case AIP-122 prefers over a generated id.
const exampleVIN = "wvwzzz1jz3w000001"

// sampleID is the example resource id for one package.
//
// 16 Crockford characters -- 80 bits -- prefixed with the resource's singular
// so the id opens with a letter, as AIP-122 requires. The vehicle is the
// exception: its identifier is its VIN.
func sampleID(pkg *model.Package) string {
	if pkg.TopLevel() && pkg.Singular == "vehicle" {
		return exampleVIN
	}

	// Seeded from the fully qualified name, so a package's example id is
	// stable across runs and distinct from its neighbours'.
	h := fnv.New64a()
	_, _ = h.Write([]byte(pkg.Root.FQN))
	n := h.Sum64()

	var b strings.Builder
	b.WriteString(pkg.Singular + "-")
	for i := 0; i < 16; i++ {
		b.WriteByte(crockford[n&31])
		// Re-hash halfway: 64 bits does not fill 16 base32 characters, and
		// repeating the first eight would be visibly patterned.
		if n >>= 5; n == 0 || i == 7 {
			h.Reset()
			_, _ = h.Write([]byte(pkg.Root.FQN + strconv.Itoa(i)))
			n = h.Sum64()
		}
	}
	return b.String()
}

// sampleName is the example resource name: the pattern with every identifier
// filled in.
func sampleName(pkg *model.Package) string {
	out := pkg.Pattern
	for p := pkg; p != nil; p = p.NameParent {
		out = strings.ReplaceAll(out, "{"+p.Singular+"}", sampleID(p))
	}
	return out
}

// sampleValue renders one field's example value as JSON.
//
// Bounded where the specification bounds it, so the example never shows a
// value the schema would reject.
func (g *Generator) sampleValue(p plan.Field, n *vspec.Node) string {
	if model.HasEnum(n) {
		values := plan.EnumValues(n)
		v := values[0]
		if len(values) > 1 {
			// The first value is often UNKNOWN, which demonstrates nothing.
			v = values[1]
		}
		return `"` + plan.EnumValueName(strings.ToUpper(snake(g.M.EnumName(n.FQN))), v.Name) + `"`
	}

	switch p.Type {
	case "bool":
		return "true"
	case "string":
		return `"` + sampleString(p.Name) + `"`
	case "google.protobuf.Timestamp":
		return `"2026-09-02T10:15:30Z"`
	case "google.protobuf.Duration":
		return `"5400s"`
	case "Date":
		return `{ "year": 2026, "month": 3, "day": 14 }`
	case "double":
		return sampleNumber(n, false)
	case "int32", "int64":
		return sampleNumber(n, true)
	}
	return "null"
}

// sampleNumber picks a number for a numeric field.
//
// Only a bound the *specification* declares is used to place it. The other
// bound a field may carry is the width restored after AIP-141 widened the
// type -- uint16 becoming int32 with a 0..65535 rule -- and that says nothing
// about plausible values: a third of the way up it makes a seat 21845 mm
// high, which is not an example, it is a distraction.
func sampleNumber(n *vspec.Node, whole bool) string {
	lo, hi := parseBound(n.Min), parseBound(n.Max)
	if lo == nil || hi == nil {
		if whole {
			return "42"
		}
		return "88.5"
	}

	// A third of the way up the declared range: inside it, and clear of both
	// bounds so the example does not read as an edge case.
	v := *lo + (*hi-*lo)/3
	if whole || v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// parseBound reads a rendered bound back as a number.
func parseBound(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// sampleString invents a plausible value for a string field from its name.
func sampleString(name string) string {
	switch {
	case strings.Contains(name, "vin"):
		return exampleVIN
	case strings.HasSuffix(name, "region_code"):
		return "SE"
	case strings.Contains(name, "uid"), strings.Contains(name, "id"):
		return "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f"
	case strings.Contains(name, "version"):
		return "1.4.2"
	}
	return fmt.Sprintf("<%s>", strings.ReplaceAll(name, "_", " "))
}

// snake renders a Pascal name as snake_case, for building an enum's constant
// prefix.
func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return b.String()
}
