// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// traps.go finds the field names where VSS vocabulary meets an AIP rule.
//
// Every trap is reported against the *planned* field rather than the source
// name. Reporting the source name would list nineteen entries that never
// change, and the one new entry a specification bump introduced would be
// invisible among them -- which is the failure this exists to prevent.

import (
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// trap is one field name that hits a known AIP rule.
type trap struct {
	owner    string // the branch the field sits on
	field    string // source name -> emitted name
	rule     string // the rule it hits
	resolved bool   // the schema already answers it
}

// findTraps collects every field whose source name hits a known AIP rule.
func findTraps(m *model.Model) []trap {
	planner := plan.New(m)

	var traps []trap
	seen := map[string]bool{}

	for _, root := range m.Roots {
		root.Walk(func(n *vspec.Node) {
			for _, c := range n.Children {
				rule := ruleFor(naming.Snake(c.Name))
				if rule == "" || seen[c.FQN+rule] {
					continue
				}
				seen[c.FQN+rule] = true

				// A field the planner rejects is a build error the caller
				// already sees; nothing to add here.
				p, err := planner.Field(n, c, false)
				if err != nil {
					continue
				}
				traps = append(traps, trap{
					owner:    n.FQN,
					field:    c.Name + " -> " + p.Name,
					rule:     rule,
					resolved: resolved(c, p, rule),
				})
			}
		})
	}
	sort.Slice(traps, func(i, j int) bool {
		if traps[i].rule != traps[j].rule {
			return traps[i].rule < traps[j].rule
		}
		return traps[i].owner < traps[j].owner
	})
	return traps
}

// resolved reports whether the schema already answers a trap.
//
// Three ways, and the first is deliberately not "the name changed". A rename
// in the catalogue is a decision someone made and wrote down; `control_unit_id`
// still ends in `_id` and is still the right name, so the record of the
// decision is what settles it, not the shape of the result.
func resolved(n *vspec.Node, p plan.Field, rule string) bool {
	if _, named := catalog.FieldRenames[n.FQN]; named {
		return true
	}
	if _, accepted := catalog.AcceptedTraps[n.FQN]; accepted {
		return true
	}
	// Cleared by rule rather than by table -- a reserved word taking its
	// `_control` suffix, or AIP-140's abbreviations.
	if ruleFor(p.Name) == "" {
		return true
	}
	// AIP-142 objects to a `_time` or `_date` suffix on a field that is not a
	// timestamp. On a field that is one, the suffix is what the rule asks for.
	return rule == suffixRule && temporalType(p.Type)
}

// temporalType reports whether a planned type is one AIP-142 accepts under a
// `_time` or `_date` name.
func temporalType(t string) bool {
	switch t {
	case "google.protobuf.Timestamp", "google.protobuf.Duration", "Date":
		return true
	}
	return false
}

// suffixRule is named because [resolved] tests for it by identity.
const suffixRule = "AIP-142 `_time`/`_date` suffix implies Timestamp"

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
		return suffixRule
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
