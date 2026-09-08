// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package describe answers one question in one place: what is the
// documentation for this declaration?
//
// # Why it is shared
//
// Every declaration this schema emits carries a comment -- api-linter
// requires it, and where the source model documents nothing the text is
// synthesised. The Markdown reference then documents the same declarations
// again.
//
// Computing that text twice is how the two disagree, and they did: the
// generated protobuf said "Forward wheel drive." where the README said
// nothing at all, for 396 enum values and 174 fields. The description is a
// property of the declaration, not of the format it is being written into,
// so it is computed once here and both renderers read it.
package describe

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// Field is the full documentation for one planned field: its description, the
// unit and VSS provenance, and a deprecation notice where there is one.
func Field(p plan.Field) string {
	doc := Summary(p)

	var meta []string
	if p.Unit != "" {
		meta = append(meta, "Unit: "+UnitSymbol(strings.TrimPrefix(p.Unit, "UNIT_"))+".")
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
	// Trimmed, not just tested. A description that opens with a blank line
	// renders as two empty comment lines in the .proto and as an empty cell
	// in the Markdown, because both read the first paragraph. Producers
	// should not build one; this makes it impossible for them to.
	if doc := strings.TrimSpace(p.Doc); doc != "" {
		return doc
	}
	return synth(p.Name)
}

// synth describes a field the source model documents by structure alone.
//
// The instance-tag machinery is the whole of it: VSS declares `instanceTag`
// and the `dimension1`/`dimension2` axes inside it without a word of prose,
// and AIP-192 requires a comment on every field regardless.
func synth(name string) string {
	switch {
	case name == "instance_tag":
		return "Which instance of this branch the values belong to."
	case strings.HasPrefix(name, "dimension"):
		return "Instance axis " + strings.TrimPrefix(name, "dimension") +
			". VSS expands a branch across each axis in turn, so the axes " +
			"together name one instance."
	default:
		return naming.Humanise(name) + "."
	}
}

// elementKind renders a VSS element constant as the word a comment uses.
func elementKind(element string) string {
	return strings.ToLower(strings.TrimPrefix(element, "ELEMENT_"))
}

// EnumValue is the documentation for one enumerant.
//
// index is its position in the source enum: the first value is the zero
// value, and where the source model opens with UNDEFINED that value *is* the
// "unspecified" AIP-126 requires rather than a second spelling of it.
func EnumValue(v sdl.EnumValue, index int) string {
	doc := v.Doc
	if doc == "" {
		if index == 0 && v.Name == "UNDEFINED" {
			doc = "Not specified."
		} else {
			doc = naming.Humanise(v.Name) + "."
		}
	}
	if source, ok := SourceSpelling(v); ok {
		doc += "\n\nThe source model spells this \"" + source + "\"."
	}
	return doc
}

// SourceSpelling returns an enumerant's spelling in the source model, where
// that differs from the sanitised name the schema uses.
func SourceSpelling(v sdl.EnumValue) (string, bool) {
	d, ok := v.Directive("vspec")
	if !ok {
		return "", false
	}
	return d.Arg("originalName")
}

// Enum is the documentation for an enum, before any AIP notes are added.
func Enum(name string, e *sdl.Def) string {
	doc := e.Doc
	if doc == "" {
		doc = name + " is an allowed-value set from the source model."
	}
	if v, ok := e.Directive("vspec"); ok {
		if fqn, ok := v.Arg("fqn"); ok {
			doc += "\n\nVSS: " + fqn + "."
		}
	}
	return doc
}

// Message is the documentation for a message, before any AIP notes are added.
func Message(name string, t *sdl.Def) string {
	if t.Doc != "" {
		return t.Doc
	}
	return name + " is a node of the COVESA Vehicle Signal Specification."
}
