// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package naming converts GraphQL SDL identifiers into protobuf ones.
//
// Every conversion runs through [SplitWords], so a rule applied there -- the
// AIP-140 abbreviation table, digit handling, acronym boundaries -- reaches
// field names, message names, enum names and enum values alike. That matters
// because api-linter checks all four: a message named Configuration fails the
// abbreviation rule exactly as a field named configuration does.
package naming

import "strings"

// SplitWords breaks a GraphQL identifier into lowercase words.
//
// It handles camelCase, underscore separation, digit runs and acronyms
// alike:
//
//	Chassis_Axle  ->  [chassis axle]
//	emissionsCo2  ->  [emissions co2]
//	ADASConfig    ->  [adas config]
//
// Each word is passed through [abbreviate] on the way out, which is what
// applies AIP-140's abbreviation table uniformly.
func SplitWords(s string) []string {
	var (
		words []string
		cur   []rune
	)
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
			if startsWord(rs, i) {
				flush()
			}
			cur = append(cur, r)
		case isDigit(r):
			// A digit run continues the current word: `Co2` is one word and
			// `Row1` is one word, which is what VSS means by both.
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

// startsWord reports whether the upper-case rune at i opens a new word.
//
// Two cases: a lower-to-upper transition (`fooBar`), and the end of an
// acronym run, where the last capital belongs to the following word -- the
// `S` in `ADASConfig` opens `config` rather than standing alone.
func startsWord(rs []rune, i int) bool {
	if i == 0 {
		return false
	}
	prev := rs[i-1]
	if isLower(prev) || isDigit(prev) {
		return true
	}
	return isUpper(prev) && i+1 < len(rs) && isLower(rs[i+1])
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// abbreviations is AIP-140's table of words that must be written short.
//
// `identifier` is deliberately absent. Shortening it gives `id`, which then
// fails AIP-148's use-uid rule; the one place it occurs is renamed instead,
// through the catalog's field and type tables.
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

// Snake renders an identifier as protobuf field case: `driverPosition` gives
// `driver_position`.
func Snake(s string) string { return strings.Join(SplitWords(s), "_") }

// Screaming renders an identifier as protobuf enum value case.
func Screaming(s string) string { return strings.ToUpper(Snake(s)) }

// Pascal renders an identifier as protobuf message case.
func Pascal(s string) string {
	var b strings.Builder
	for _, w := range SplitWords(s) {
		b.WriteString(strings.ToUpper(w[:1]))
		b.WriteString(w[1:])
	}
	return b.String()
}

// LowerCamel renders an identifier as a resource-name segment, which AIP-122
// writes in camelCase: `ControlUnit` gives `controlUnit`.
func LowerCamel(s string) string {
	p := Pascal(s)
	if p == "" {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// ContainsWord reports whether a split identifier contains the given word.
func ContainsWord(words []string, w string) bool {
	for _, x := range words {
		if x == w {
			return true
		}
	}
	return false
}

// Humanise turns an identifier into a sentence fragment for a comment.
func Humanise(v string) string {
	words := SplitWords(v)
	if len(words) == 0 {
		return v
	}
	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	return strings.Join(words, " ")
}
