// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package describe answers one question in one place: what is the
// documentation for this declaration?
//
// # Why it is shared
//
// Every declaration this schema emits carries a comment — api-linter requires
// it, and where the specification documents nothing the text is synthesised.
// The Markdown reference then documents the same declarations again.
//
// Computing that text twice is how the two disagree, and they did: the
// generated protobuf said "Forward wheel drive." where the README said
// nothing at all, for 396 enum values and 174 fields. The description is a
// property of the declaration, not of the format it is written into, so it is
// computed once here and both renderers read it.
package describe

import (
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// Field is the full documentation for one planned field: its description, the
// specification's own longer note, the unit and provenance, and a deprecation
// notice where there is one.
func Field(p plan.Field, symbol string) string {
	doc := Summary(p)

	// VSS attaches a `comment` where behaviour needs explaining beyond the
	// one-line description. 181 signals carry one, and the GraphQL
	// translation this generator used to read dropped every one of them.
	if p.Comment != "" {
		doc += "\n\n" + p.Comment
	}

	// A decision this schema made rather than one it read. It sits with the
	// description because a consumer meeting the field has no other place to
	// learn why its type is not the one the source model declares.
	if p.Note != "" {
		doc += "\n\n" + p.Note
	}

	var meta []string
	if symbol != "" {
		meta = append(meta, "Unit: "+symbol+".")
	}
	if p.FQN != "" {
		if kind := elementKind(p.Element); kind != "" {
			meta = append(meta, "VSS: "+p.FQN+" ("+kind+").")
		} else {
			meta = append(meta, "VSS: "+p.FQN+".")
		}
	}
	if len(meta) > 0 {
		if doc != "" {
			doc += "\n\n"
		}
		doc += strings.Join(meta, " ")
	}

	// AIP-192 requires "Deprecated: <reason>" to be the *first* line of the
	// comment, so it goes above the description rather than after it.
	if p.Deprecated != "" {
		doc = "Deprecated: " + p.Deprecated + "\n\n" + doc
	}
	return doc
}

// Summary is a field's description alone, without the unit and provenance
// lines Field appends.
//
// This is what a table cell wants: the reference already has columns for the
// unit and the behaviour, so repeating them in the description would be
// noise.
func Summary(p plan.Field) string {
	// Trimmed, not just tested. A description opening with a blank line
	// renders as two empty comment lines in the .proto and as an empty cell
	// in the Markdown, because both read the first paragraph.
	if doc := strings.TrimSpace(p.Doc); doc != "" {
		return doc
	}
	return synth(p.Name)
}

// synth describes a field the specification documents by structure alone.
//
// One case rather than the several this had while the generator read VSS
// through its GraphQL translation. That translation modelled an instance as a
// message of dimension fields; VSS itself expands the path instead, so a seat
// is a resource of its own and there is no axis field left to describe.
func synth(name string) string {
	if name == "instance_tag" {
		return "Which instance of this branch the values belong to."
	}
	return naming.Humanise(name) + "."
}

// elementKind renders a VSS element constant as the word a comment uses.
func elementKind(element string) string {
	return strings.ToLower(strings.TrimPrefix(element, "ELEMENT_"))
}

// EnumValue is the documentation for one enumerant.
//
// index is its position: the first value is the zero value, and where the
// specification opens with UNKNOWN that value *is* the "unspecified" AIP-126
// requires rather than a second spelling of it.
func EnumValue(name string, index int) string {
	if index == 0 && (name == "UNKNOWN" || name == "UNDEFINED") {
		return "Not specified."
	}
	return naming.Humanise(name) + "."
}

// Enum is the documentation for an enum, before any AIP notes are added.
func Enum(name string, n *vspec.Node) string {
	doc := n.Description
	if doc == "" {
		doc = name + " is an allowed-value set from the specification."
	}
	return doc + "\n\nVSS: " + n.FQN + "."
}

// Message is the documentation for a message, before any AIP notes.
func Message(name string, n *vspec.Node) string {
	doc := n.Description
	if doc == "" {
		doc = name + " is a node of the COVESA Vehicle Signal Specification."
	}
	if n.Comment != "" {
		doc += "\n\n" + n.Comment
	}
	return doc
}
