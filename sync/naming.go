// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// naming.go converts GraphQL SDL identifiers into protobuf ones.

import "strings"

// prepositions AIP-140 bans from the start of a field name.
var prepositions = []string{
	"and_", "at_", "before_", "after_", "by_", "for_", "from_", "in_", "of_",
	"on_", "or_", "to_", "with_",
}

// hasPreposition reports whether a snake_case name begins with a preposition
// AIP-140 forbids.
func hasPreposition(n string) bool {
	for _, p := range prepositions {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

// timeUnits are the bare unit names AIP-142 reads as a Timestamp field.
var timeUnits = map[string]bool{
	"seconds": true, "minutes": true, "hours": true, "days": true,
	"weeks": true, "months": true, "years": true, "time": true, "date": true,
}

// bareTimeUnit reports whether a name is a bare time unit.
func bareTimeUnit(n string) bool { return timeUnits[n] }

// splitWords breaks a GraphQL identifier into lowercase words, handling
// camelCase, underscore separation, digit runs and acronyms alike: `Chassis_Axle`
// gives [chassis axle], `emissionsCo2` gives [emissions co2], and
// `ADASConfig` gives [adas config].
func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, abbreviate(strings.ToLower(string(cur))))
			cur = nil
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r == '_' || r == '-' || r == ' ':
			flush()
		case isUpper(r):
			// Start a new word at a lower-to-upper transition, and at the end
			// of an acronym run (the `S` in `ADASConfig` opens `config`, not
			// a word of its own).
			if i > 0 && (isLower(rs[i-1]) || isDigit(rs[i-1]) ||
				(isUpper(rs[i-1]) && i+1 < len(rs) && isLower(rs[i+1]))) {
				flush()
			}
			cur = append(cur, r)
		case isDigit(r):
			// A digit run continues the current word: `Co2` is one word, and
			// `Row1` is one word, which is what VSS means by both.
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// abbreviations is AIP-140's table of words that must be written short.
// Applied to every identifier -- field, message, enum and enum value alike --
// because the rule checks all four and a message named Configuration fails it
// exactly as a field named configuration does.
//
// `identifier` is deliberately absent. Shortening it gives `id`, which then
// fails AIP-148's use-uid rule; the one place it occurs is renamed instead,
// through typeRenames and fieldRenames.
var abbreviations = map[string]string{
	"configuration": "config",
	"information":   "info",
	"specification": "spec",
	"statistics":    "stats",
}

// abbreviate applies the AIP-140 table to one lowercase word.
func abbreviate(w string) string {
	if a, ok := abbreviations[w]; ok {
		return a
	}
	return w
}

// snake renders an identifier as protobuf field case.
func snake(s string) string { return strings.Join(splitWords(s), "_") }

// screaming renders an identifier as protobuf enum value case.
func screaming(s string) string { return strings.ToUpper(snake(s)) }

// pascal renders an identifier as protobuf message case.
func pascal(s string) string {
	var b strings.Builder
	for _, w := range splitWords(s) {
		b.WriteString(strings.ToUpper(w[:1]))
		b.WriteString(w[1:])
	}
	return b.String()
}

// lowerCamel renders an identifier as a resource-name collection segment,
// which AIP-122 writes in camelCase: `ControlUnit` gives `controlUnits`.
func lowerCamel(s string) string {
	p := pascal(s)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// irregularPlurals are the branch names English does not pluralise by adding
// an "s". Each is a real VSS branch name; the table is not speculative.
//
// A plural is not cosmetic here: AIP-123 ties the resource's `plural` to the
// collection segment of its resource name and to its service name, so
// "Chassiss" would appear in every URL and every generated client.
var irregularPlurals = map[string]string{
	// Already plural, or a mass noun that takes no plural. VSS uses each as
	// the name of one branch.
	"adas":                  "adas",
	"diagnostics":           "diagnostics",
	"connectivity":          "connectivity",
	"exterior":              "exterior",
	"powertrain":            "powertrain",
	"motionManagement":      "motionManagement",
	"vehicleIdentification": "vehicleIdentification",
	"versionVss":            "versionVss",
	"trailer":               "trailers",
	"acceleration":          "accelerations",

	// Ends in a sibilant, so the plural takes "es".
	"chassis": "chassisSet",

	// Irregular in English.
	"person": "people",

	// Already plural in the source model: VSS names the branch `Mirrors`
	// even though it describes one mirror assembly.
	"mirrors": "mirrors",
}

// pluralise renders the AIP-123 plural of a lowerCamel resource singular.
func pluralise(singular string) string {
	if p, ok := irregularPlurals[singular]; ok {
		return p
	}
	switch {
	case strings.HasSuffix(singular, "y") && !hasVowelBefore(singular, "y"):
		// velocity -> velocities
		return singular[:len(singular)-1] + "ies"
	case strings.HasSuffix(singular, "s"), strings.HasSuffix(singular, "x"),
		strings.HasSuffix(singular, "z"), strings.HasSuffix(singular, "ch"),
		strings.HasSuffix(singular, "sh"):
		return singular + "es"
	default:
		return singular + "s"
	}
}

// hasVowelBefore reports whether the character before the given suffix is a
// vowel, which decides between "-ys" and "-ies".
func hasVowelBefore(s, suffix string) bool {
	i := len(s) - len(suffix)
	if i <= 0 {
		return false
	}
	return strings.ContainsRune("aeiou", rune(s[i-1]))
}
