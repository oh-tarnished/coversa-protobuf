// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package vspec

// enum.go reads VSS's explicitly numbered allowed-value sets.
//
// VSS has two ways to constrain a value. `allowed` is a list of names; `enum`
// is a mapping of name to number:
//
//	enum:
//	  UNKNOWN: 0
//	  DRY: 1
//
// The second is the richer of the two and the GraphQL translation cannot
// express it at all. The numbers are the specification's own, so a protobuf
// enum adopts them rather than inventing a sequence.

import "gopkg.in/yaml.v3"

// valueFor returns the value node for a key of a mapping, or nil.
func valueFor(m *yaml.Node, key string) *yaml.Node {
	if m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// enumEntries reads an explicitly numbered enum, preserving declaration order.
//
// A map would lose that order, and the order is how a reader finds a value in
// the emitted protobuf; the numbers are VSS's own and are carried through
// rather than reassigned.
func enumEntries(m *yaml.Node) []EnumEntry {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	out := make([]EnumEntry, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		var number int
		if err := m.Content[i+1].Decode(&number); err != nil {
			continue
		}
		out = append(out, EnumEntry{Name: m.Content[i].Value, Number: number})
	}
	return out
}
