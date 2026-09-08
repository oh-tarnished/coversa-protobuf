// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// field.go writes one field: its comment, its type, and the annotations that
// say what it is in VSS terms.
import (
	"fmt"
	"strings"
)

// renderReference writes a field that names a resource in another package.
func renderReference(m *Model, owner *Def, fl Field, other *Package, num int, sb *strings.Builder) {
	name := snake(fl.Name)
	if r, ok := fieldRenames[owner.Name+"."+fl.Name]; ok {
		name = r
	}
	res := messageName(other.Root.Name)
	doc := fl.Doc
	if strings.TrimSpace(doc) == "" {
		doc = "The " + res + " this " + messageName(owner.Name) + " refers to."
	}
	doc += "\n\nThe resource name, \"" + other.Pattern + "\", not the " + res +
		" itself: AIP-215 <https://aip.dev/215> forbids a field naming a message " +
		"in another proto package. Resolve it with " + other.ServiceName() + ".Get" +
		res + "."
	sb.WriteString(docBlock(doc, "  "))

	behavior := "OPTIONAL"
	if fl.Type.NonNull {
		behavior = "REQUIRED"
	}
	sb.WriteString("  string " + name + " = " + itoa(num) + " [\n")
	sb.WriteString("    (google.api.field_behavior) = " + behavior + ",\n")
	if fl.Type.NonNull {
		// A required association to a past event does not change afterwards.
		sb.WriteString("    (google.api.field_behavior) = IMMUTABLE,\n")
	}
	sb.WriteString("    (google.api.resource_reference) = {type: \"vdm.covesa.org/" + res + "\"}\n")
	sb.WriteString("  ];\n\n")
}

// renderResourceHeader writes the google.api.resource option and the AIP
// identity and lifecycle fields every resource here carries.
func renderResourceHeader(pkg *Package, sb *strings.Builder) {
	sb.WriteString("  option (google.api.resource) = {\n")
	sb.WriteString("    type: \"vdm.covesa.org/" + messageName(pkg.Root.Name) + "\"\n")
	sb.WriteString("    pattern: \"" + pkg.Pattern + "\"\n")
	sb.WriteString("    singular: \"" + pkg.Singular + "\"\n")
	sb.WriteString("    plural: \"" + pkg.Plural + "\"\n")
	sb.WriteString("  };\n\n")

	pat := patternToRegex(pkg.Pattern)
	sb.WriteString(docBlock("Resource name, \""+pkg.Pattern+"\". Assigned by the server.", "  "))
	sb.WriteString("  string name = 1 [\n")
	sb.WriteString("    (google.api.field_behavior) = IDENTIFIER,\n")
	sb.WriteString("    (buf.validate.field).string.pattern = \"" + pat + "\",\n")
	sb.WriteString("    (buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE\n")
	sb.WriteString("  ];\n\n")

	sb.WriteString(docBlock("Server-assigned unique id.", "  "))
	sb.WriteString("  string uid = 2 [\n")
	sb.WriteString("    (google.api.field_behavior) = OUTPUT_ONLY,\n")
	sb.WriteString("    (google.api.field_info).format = UUID4,\n")
	sb.WriteString("    (buf.validate.field).string.uuid = true,\n")
	sb.WriteString("    (buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE\n")
	sb.WriteString("  ];\n\n")

	// Field numbers 8 to 15 are held for AIP identity and lifecycle fields a
	// later revision may add, so adding one never renumbers a signal.
	//
	// Reserved rather than merely skipped. A serialization target numbers its
	// slots contiguously from zero, so an unreserved hole is not a hole at
	// all -- every field after it slides down one slot, and a consumer
	// compiled against the old schema reads the wrong one. `reserved` keeps
	// the slots occupied. protoc-gen-buffers rejects the schema without it.
	sb.WriteString("  reserved 8 to 15;\n\n")

	lifecycle := []struct{ name, doc string }{
		{"etag", "Weak validator for optimistic concurrency, AIP-154 <https://aip.dev/154>."},
		{"delete_time", "When this resource was soft-deleted, unset while it is live. AIP-164 <https://aip.dev/164>."},
		{"expire_time", "When a soft-deleted resource is purged for good. AIP-164 <https://aip.dev/164>."},
		{"create_time", "When this resource was created."},
		{"update_time", "When this resource was last modified."},
	}
	for i, l := range lifecycle {
		sb.WriteString(docBlock(l.doc, "  "))
		typ := "google.protobuf.Timestamp"
		if l.name == "etag" {
			typ = "string"
		}
		sb.WriteString(fmt.Sprintf("  %s %s = %d [(google.api.field_behavior) = OUTPUT_ONLY];\n\n", typ, l.name, i+3))
	}
}

// renderField writes one field with its comment and annotations.
func renderField(p FieldPlan, num int, sb *strings.Builder) {
	doc := p.Doc
	// AIP-192 requires a comment on every field. The source model leaves the
	// instance-tag machinery undocumented -- `instanceTag` and the
	// `dimension1`/`dimension2` axes inside it -- so a description is
	// synthesised from what the field is rather than left blank.
	if strings.TrimSpace(doc) == "" {
		doc = synthDoc(p.Name, p.Type)
	}
	var meta []string
	if p.Unit != "" {
		meta = append(meta, "Unit: "+unitSymbol(p.Unit)+".")
	}
	if p.FQN != "" {
		kind := strings.ToLower(strings.TrimPrefix(p.Element, "ELEMENT_"))
		if kind == "" {
			meta = append(meta, "VSS: "+p.FQN+".")
		} else {
			meta = append(meta, "VSS: "+p.FQN+" ("+kind+").")
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
	sb.WriteString(docBlock(doc, "  "))

	prefix := "  "
	if p.Repeated {
		prefix += "repeated "
	}
	var opts []string
	for _, b := range p.Behavior {
		opts = append(opts, "(google.api.field_behavior) = "+b)
	}
	opts = append(opts, p.Validate...)
	if p.Element != "" {
		var inner []string
		inner = append(inner, "      element: "+p.Element)
		if p.FQN != "" {
			inner = append(inner, "      fqn: \""+p.FQN+"\"")
		}
		if p.Unit != "" {
			inner = append(inner, "      unit: "+p.Unit)
			inner = append(inner, "      quantity_kind: "+p.Quantity)
		}
		opts = append(opts, "("+vocabPackage+".signal) = {\n"+strings.Join(inner, "\n")+"\n    }")
	}
	if p.Deprecated != "" {
		opts = append(opts, "deprecated = true")
	}
	// A lone single-line option sits on the field's own line; buf format
	// collapses it there anyway, and emitting it pre-collapsed keeps the
	// generator's raw output identical to the formatted tree.
	if len(opts) == 1 && !strings.Contains(opts[0], "\n") {
		sb.WriteString(fmt.Sprintf("%s%s %s = %d [%s];\n\n",
			prefix, p.Type, p.Name, num, opts[0]))
		return
	}
	sb.WriteString(fmt.Sprintf("%s%s %s = %d [\n    %s\n  ];\n\n",
		prefix, p.Type, p.Name, num, strings.Join(opts, ",\n    ")))
}

// synthDoc describes a field the source model documents by structure alone.
func synthDoc(name, typ string) string {
	switch {
	case name == "instance_tag":
		return "Which instance of this branch the values belong to."
	case strings.HasPrefix(name, "dimension"):
		n := strings.TrimPrefix(name, "dimension")
		return "Instance axis " + n + ". VSS expands a branch across each axis " +
			"in turn, so the axes together name one instance."
	default:
		return humanise(name) + "."
	}
}
