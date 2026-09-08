// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package naming

import "strings"

// irregularPlurals are the branch names English does not pluralise by adding
// an "s". Each is a real VSS branch name; the table is not speculative.
//
// A plural is not cosmetic here. AIP-123 ties a resource's `plural` to the
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

	// Ends in a sibilant, and "chassises" is not a word anyone writes.
	"chassis": "chassisSet",

	// Irregular in English.
	"person": "people",

	// Already plural in the source model: VSS names the branch `Mirrors`
	// even though it describes one mirror assembly.
	"mirrors": "mirrors",
}

// Pluralise renders the AIP-123 plural of a lowerCamel resource singular.
//
// The regular cases are handled by rule so a specification bump does not need
// a table entry for every new branch; only the words English is irregular
// about are listed above.
func Pluralise(singular string) string {
	if p, ok := irregularPlurals[singular]; ok {
		return p
	}
	switch {
	case strings.HasSuffix(singular, "y") && !endsVowelY(singular):
		// velocity -> velocities, but valley -> valleys
		return singular[:len(singular)-1] + "ies"
	case hasAnySuffix(singular, "s", "x", "z", "ch", "sh"):
		return singular + "es"
	default:
		return singular + "s"
	}
}

// endsVowelY reports whether a name ends in a vowel followed by "y", which
// decides between "-ys" and "-ies".
func endsVowelY(s string) bool {
	if len(s) < 2 {
		return false
	}
	return strings.ContainsRune("aeiou", rune(s[len(s)-2]))
}

// hasAnySuffix reports whether s ends in any of the given suffixes.
func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}
