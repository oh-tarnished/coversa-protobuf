// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// resource.go writes the AIP identity and lifecycle fields every resource
// message carries, and the enums a file holds.

import (
	"fmt"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/internal/catalog"
	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/naming"
	"github.com/the-protobuf-project/vdm/sync/internal/plan"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// lifecycleFields are the server-owned fields every resource carries, in
// field-number order starting at 3.
var lifecycleFields = []struct{ name, doc string }{
	{"etag", "Weak validator for optimistic concurrency, AIP-154 <https://aip.dev/154>."},
	{"delete_time", "When this resource was soft-deleted, unset while it is live. AIP-164 <https://aip.dev/164>."},
	{"expire_time", "When a soft-deleted resource is purged for good. AIP-164 <https://aip.dev/164>."},
	{"create_time", "When this resource was created."},
	{"update_time", "When this resource was last modified."},
}

// renderResourceHeader writes the google.api.resource option and the AIP
// identity and lifecycle fields.
func (e *Emitter) renderResourceHeader(pkg *model.Package, sb *strings.Builder) {
	res := pkg.ResourceName()

	sb.WriteString("  option (google.api.resource) = {\n")
	sb.WriteString("    type: \"vdm.covesa.org/" + res + "\"\n")
	sb.WriteString("    pattern: \"" + pkg.Pattern + "\"\n")
	sb.WriteString("    singular: \"" + pkg.Singular + "\"\n")
	sb.WriteString("    plural: \"" + pkg.Plural + "\"\n")
	sb.WriteString("  };\n\n")

	sb.WriteString(docBlock("Resource name, \""+pkg.Pattern+"\". Assigned by the server.", "  "))
	sb.WriteString("  string name = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = IDENTIFIER,\n")
	sb.WriteString("    (buf.validate.field).string.pattern = \"" + model.PatternRegex(pkg.Pattern) + "\",\n")
	sb.WriteString("    (buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE\n")
	sb.WriteString("  ];\n\n")

	sb.WriteString(docBlock("Server-assigned unique id.", "  "))
	sb.WriteString("  string uid = 2 [\n")
	sb.WriteString("    (google.api.field_behavior) = OUTPUT_ONLY,\n")
	sb.WriteString("    (google.api.field_info).format = UUID4,\n")
	sb.WriteString("    (buf.validate.field).string.uuid = true,\n")
	sb.WriteString("    (buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE\n")
	sb.WriteString("  ];\n\n")

	// Field numbers 8 to 15 are held for AIP fields a later revision may add,
	// so adding one never renumbers a signal.
	//
	// Reserved rather than merely skipped. A serialization target numbers its
	// slots contiguously from zero, so an unreserved hole is not a hole at
	// all -- every field after it slides down one slot, and a consumer
	// compiled against the old schema reads the wrong one.
	sb.WriteString("  reserved 8 to 15;\n\n")

	for i, l := range lifecycleFields {
		sb.WriteString(docBlock(l.doc, "  "))
		typ := "google.protobuf.Timestamp"
		if l.name == "etag" {
			typ = "string"
		}
		fmt.Fprintf(sb, "  %s %s = %d [(google.api.field_behavior) = OUTPUT_ONLY];\n\n",
			typ, l.name, i+3)
	}
}

// renderEnum writes one enum, its values prefixed with the enum name so they
// do not collide in the package scope proto3 gives them.
func (e *Emitter) renderEnum(f *File, enum *sdl.Def, sb *strings.Builder) {
	name := plan.EnumName(enum.Name)
	prefix := naming.Screaming(name)

	doc := enum.Doc
	if doc == "" {
		doc = name + " is an allowed-value set from the source model."
	}
	if v, ok := enum.Directive("vspec"); ok {
		if fqn, ok := v.Arg("fqn"); ok {
			doc += "\n\nVSS: " + fqn + "."
		}
	}
	sb.WriteString(docBlock(doc, ""))
	sb.WriteString("enum " + name + " {\n")

	// AIP-126 requires a zero value meaning "unspecified". Where the source
	// model already opens with UNDEFINED, that value *is* the zero rather
	// than a second spelling of it.
	num := 0
	if len(enum.Values) == 0 || enum.Values[0].Name != "UNDEFINED" {
		sb.WriteString("  // Not specified.\n")
		sb.WriteString("  " + prefix + "_UNSPECIFIED = 0;\n\n")
		num = 1
	}
	for i, v := range enum.Values {
		e.renderEnumValue(f, v, prefix, i, num, sb)
		num++
	}
	trimTrailingBlank(sb)
	sb.WriteString("}\n")
}

// renderEnumValue writes one enumerant, recording the source spelling where
// the sanitised name differs from it.
func (e *Emitter) renderEnumValue(f *File, v sdl.EnumValue, prefix string, index, num int, sb *strings.Builder) {
	source, hasSource := "", false
	if d, ok := v.Directive("vspec"); ok {
		if s, ok := d.Arg("originalName"); ok {
			source, hasSource = s, true
		}
	}

	doc := v.Doc
	if doc == "" {
		if index == 0 && v.Name == "UNDEFINED" {
			doc = "Not specified."
		} else {
			doc = naming.Humanise(v.Name) + "."
		}
	}
	if hasSource {
		doc += "\n\nThe source model spells this \"" + source + "\"."
	}
	sb.WriteString(docBlock(doc, "  "))

	value := prefix + "_" + naming.Screaming(v.Name) + catalog.EnumValueSuffix(v.Name)
	if index == 0 && v.Name == "UNDEFINED" {
		value = prefix + "_UNSPECIFIED"
	}
	if hasSource {
		f.Imports[model.VocabRoot+"/annotations.proto"] = true
		fmt.Fprintf(sb, "  %s = %d [(%s.allowed_value) = {source_spelling: %q}];\n\n",
			value, num, model.VocabPackage, source)
		return
	}
	fmt.Fprintf(sb, "  %s = %d;\n\n", value, num)
}
