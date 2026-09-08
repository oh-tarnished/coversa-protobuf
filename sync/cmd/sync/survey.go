// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// survey.go reports what the parser found and every field name that trips a
// known AIP naming rule.
//
// This is how the rename catalogue in docs/conventions.md was built, and it
// is kept so the catalogue can be *rechecked* after a specification bump
// rather than trusted. A new signal name may hit a rule no existing field
// does; this is what surfaces it before the linter does.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// trap is one field name that hits a known AIP rule.
type trap struct{ owner, field, rule string }

// surveyModel prints the parsed model and its naming traps.
func surveyModel(defs []sdl.Def) error {
	printKinds(defs)

	traps := findTraps(defs)
	fmt.Printf("\nAIP naming traps: %d\n", len(traps))
	for _, t := range traps {
		fmt.Printf("  %-40s %-46s %s\n", t.owner, t.field, t.rule)
	}
	return nil
}

// printKinds reports how many definitions of each kind were parsed.
func printKinds(defs []sdl.Def) {
	counts := map[string]int{}
	for _, d := range defs {
		counts[d.Kind]++
	}
	kinds := make([]string, 0, len(counts))
	for k := range counts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Printf("%-14s %d\n", k, counts[k])
	}
}

// findTraps collects every field whose name hits a known AIP rule.
func findTraps(defs []sdl.Def) []trap {
	var traps []trap
	seen := map[string]bool{}

	for _, d := range defs {
		for _, f := range d.Fields {
			n := naming.Snake(f.Name)
			rule := ruleFor(n)
			if rule == "" {
				continue
			}
			if key := d.Name + "." + n + rule; !seen[key] {
				seen[key] = true
				traps = append(traps, trap{d.Name, f.Name + " -> " + n, rule})
			}
		}
	}
	sort.Slice(traps, func(i, j int) bool {
		if traps[i].rule != traps[j].rule {
			return traps[i].rule < traps[j].rule
		}
		return traps[i].owner < traps[j].owner
	})
	return traps
}

// ruleFor names the AIP rule a snake_case field name trips, or "".
func ruleFor(n string) string {
	switch {
	case n == "name":
		return "AIP-122 bare `name` marks the message a resource"
	case strings.HasSuffix(n, "_name") && !allowedNameSuffix[n]:
		return "AIP-122 `_name` suffix"
	case n == "state" || n == "status":
		return "AIP-216 reserved"
	case hasPreposition(n):
		return "AIP-140 preposition"
	case bareTimeUnits[n]:
		return "AIP-142 bare time unit reads as a Timestamp"
	case strings.HasSuffix(n, "_time") || strings.HasSuffix(n, "_date"):
		return "AIP-142 `_time`/`_date` suffix implies Timestamp"
	case strings.HasSuffix(n, "_id"):
		return "AIP-122 `_id` suffix"
	default:
		return ""
	}
}

// allowedNameSuffix are the three fields AIP-122 lets keep a `_name` suffix.
var allowedNameSuffix = map[string]bool{
	"display_name": true, "given_name": true, "family_name": true,
}

// bareTimeUnits are the names AIP-142 reads as a Timestamp field.
var bareTimeUnits = map[string]bool{
	"seconds": true, "minutes": true, "hours": true, "days": true,
	"weeks": true, "months": true, "years": true, "time": true, "date": true,
}

// prepositions AIP-140 bans from the start of a field name.
var prepositions = []string{
	"and_", "at_", "before_", "after_", "by_", "for_", "from_", "in_", "of_",
	"on_", "or_", "to_", "with_",
}

// hasPreposition reports whether a name begins with a banned preposition.
func hasPreposition(n string) bool {
	for _, p := range prepositions {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}
