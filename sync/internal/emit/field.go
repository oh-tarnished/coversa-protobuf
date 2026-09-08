// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// field.go writes one field: its comment, its type, and the annotations that
// say what it is in VSS terms.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/catalog"
	"github.com/the-protobuf-project/vdm/sync/describe"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// recordImports notes every file a planned field's type pulls in.
func (e *Emitter) recordImports(pkg *model.Package, f *File, p plan.Field, src sdl.Field, fileOf map[string]string) {
	for _, i := range p.Imports {
		f.Imports[i] = true
	}
	if len(p.Validate) > 0 {
		f.Imports["buf/validate/validate.proto"] = true
	}
	if p.Element != "" {
		f.Imports[model.VocabRoot+"/annotations.proto"] = true
	}
	// The import usually follows the source field's type name, but a field
	// promoted to another message -- a date string becoming a Date -- names a
	// type the source field never mentioned. Resolve on the planned type,
	// falling back to the source name.
	dep, ok := fileOf[p.Type]
	if !ok {
		dep, ok = fileOf[src.Type.Name]
	}
	if ok && dep != f.Base {
		f.Imports[pkg.ImportPath(dep)] = true
	}
}

// renderField writes one field with its comment and annotations.
func renderField(p plan.Field, num int, sb *strings.Builder) {
	sb.WriteString(docBlock(describe.Field(p), "  "))

	prefix := "  "
	if p.Repeated {
		prefix += "repeated "
	}
	opts := fieldOptions(p)

	// A lone single-line option sits on the field's own line; buf format
	// collapses it there anyway, and emitting it pre-collapsed keeps the raw
	// output identical to the formatted tree.
	if len(opts) == 1 && !strings.Contains(opts[0], "\n") {
		fmt.Fprintf(sb, "%s%s %s = %d [%s];\n\n", prefix, p.Type, p.Name, num, opts[0])
		return
	}
	fmt.Fprintf(sb, "%s%s %s = %d [\n    %s\n  ];\n\n",
		prefix, p.Type, p.Name, num, strings.Join(opts, ",\n    "))
}

// fieldOptions renders the option list a field carries.
func fieldOptions(p plan.Field) []string {
	var opts []string
	for _, b := range p.Behavior {
		opts = append(opts, "(google.api.field_behavior) = "+b)
	}
	opts = append(opts, p.Validate...)

	if p.Element != "" {
		inner := []string{"      element: " + p.Element}
		if p.FQN != "" {
			inner = append(inner, "      fqn: \""+p.FQN+"\"")
		}
		if p.Unit != "" {
			inner = append(inner,
				"      unit: "+p.Unit,
				"      quantity_kind: "+p.Quantity)
		}
		opts = append(opts, "("+model.VocabPackage+".signal) = {\n"+
			strings.Join(inner, "\n")+"\n    }")
	}
	if p.Deprecated != "" {
		opts = append(opts, "deprecated = true")
	}
	return opts
}

// renderReference writes a field that names a resource in another package.
func (e *Emitter) renderReference(owner *sdl.Def, src sdl.Field, other *model.Package, num int, sb *strings.Builder) {
	name := naming.Snake(src.Name)
	if r, ok := catalog.FieldRenames[owner.Name+"."+src.Name]; ok {
		name = r
	}
	res := other.ResourceName()

	doc := src.Doc
	if strings.TrimSpace(doc) == "" {
		doc = "The " + res + " this " + model.MessageName(owner.Name) + " refers to."
	}
	doc += "\n\nThe resource name, \"" + other.Pattern + "\", not the " + res +
		" itself: AIP-215 <https://aip.dev/215> forbids a field naming a message " +
		"in another proto package. Resolve it with " + other.ServiceName() + ".Get" +
		res + "."
	sb.WriteString(docBlock(doc, "  "))

	behavior := "OPTIONAL"
	if src.Type.NonNull {
		behavior = "REQUIRED"
	}
	sb.WriteString("  string " + name + " = " + strconv.Itoa(num) + " [\n")
	sb.WriteString("    (google.api.field_behavior) = " + behavior + ",\n")
	if src.Type.NonNull {
		// A required association to a past event does not change afterwards.
		sb.WriteString("    (google.api.field_behavior) = IMMUTABLE,\n")
	}
	sb.WriteString("    (google.api.resource_reference) = {type: \"vdm.covesa.org/" + res + "\"}\n")
	sb.WriteString("  ];\n\n")
}
