// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// message.go renders one file: its header, its message, and the enums that
// travel with it.

import (
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// firstSignalFieldNumber leaves 8-15 free below it for AIP identity fields a
// future revision may add, so adding one never renumbers a signal.
const firstSignalFieldNumber = 16

// renderFile produces the complete text of one file.
func (e *Emitter) renderFile(pkg *model.Package, f *File, fileOf map[string]string) (string, error) {
	var body strings.Builder
	if !f.TypesOnly {
		if err := e.renderMessage(pkg, f, &body, fileOf); err != nil {
			return "", err
		}
	}
	for _, enum := range f.Enums {
		e.renderEnum(f, enum, &body)
	}
	if f.EnumFile != "" {
		f.Imports[pkg.ImportPath(f.EnumFile)] = true
	}
	return e.header(pkg, f) + body.String(), nil
}

// header renders the banner, package clause, imports and Java options.
func (e *Emitter) header(pkg *model.Package, f *File) string {
	var sb strings.Builder
	sb.WriteString(e.M.Spec.Banner())
	sb.WriteString("syntax = \"proto3\";\n\n")
	sb.WriteString("package " + pkg.ProtoPackage() + ";\n\n")

	imports := make([]string, 0, len(f.Imports))
	for i := range f.Imports {
		imports = append(imports, i)
	}
	sort.Strings(imports)
	for _, i := range imports {
		sb.WriteString("import \"" + i + "\";\n")
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}

	sb.WriteString("option java_multiple_files = true;\n")
	sb.WriteString("option java_outer_classname = \"" + model.OuterClassname(f.Base) + "\";\n")
	sb.WriteString("option java_package = \"" + pkg.JavaPackage() + "\";\n\n")
	return sb.String()
}

// renderMessage writes one message, as a resource when it is the package root.
func (e *Emitter) renderMessage(pkg *model.Package, f *File, sb *strings.Builder, fileOf map[string]string) error {
	t := f.Type
	f.Imports["google/api/field_behavior.proto"] = true

	sb.WriteString(docBlock(e.messageDoc(pkg, t, f.Root), ""))
	sb.WriteString("message " + model.MessageName(t.Name) + " {\n")

	if f.Root {
		for _, i := range resourceImports {
			f.Imports[i] = true
		}
		e.renderResourceHeader(pkg, sb)
	}
	e.renderBranchOption(f, t, sb)

	num := 1
	if f.Root {
		num = firstSignalFieldNumber
	}
	for _, child := range t.Children {
		written, err := e.renderOneField(pkg, f, t, child, num, sb, fileOf)
		if err != nil {
			return err
		}
		num += written
	}
	// Close immediately after the last member: buf format would remove the
	// blank line, and emitting output the formatter has to correct makes the
	// generator diff against its own previous run.
	trimTrailingBlank(sb)
	sb.WriteString("}\n")
	return nil
}

// resourceImports are the files every resource message needs.
var resourceImports = []string{
	"google/api/resource.proto",
	"google/api/field_info.proto",
	"buf/validate/validate.proto",
	"google/protobuf/timestamp.proto",
}

// renderBranchOption writes the VSS branch annotation, when the type has one.
func (e *Emitter) renderBranchOption(f *File, t *vspec.Node, sb *strings.Builder) {
	f.Imports[model.VocabRoot+"/annotations.proto"] = true
	sb.WriteString("  option (" + model.VocabPackage + ".branch) = {\n")
	sb.WriteString("    element: ELEMENT_" + strings.ToUpper(string(t.Kind)) + "\n")
	sb.WriteString("    fqn: \"" + t.FQN + "\"\n")
	sb.WriteString("  };\n\n")
}

// renderOneField writes a single field and reports how many numbers it used.
//
// A field whose type lives in another package cannot be named directly:
// AIP-215 forbids it. There are two cases and they resolve differently -- a
// child resource is dropped, because its name is derivable from this one; a
// genuine association becomes the resource name.
func (e *Emitter) renderOneField(pkg *model.Package, f *File, owner, child *vspec.Node, num int, sb *strings.Builder, fileOf map[string]string) (int, error) {
	// A branch promoted to a resource of its own is reachable by a name
	// derived from this one, so restating it as a field would be redundant
	// and could disagree with the real name.
	if child.Kind == vspec.KindBranch && child.Ref == "" && !pkg.Holds(child.FQN) {
		return 0, nil
	}

	p, err := e.Planner.Field(owner, child, f.Root)
	if err != nil {
		return 0, err
	}
	if p.Ref != "" {
		f.Imports["google/api/resource.proto"] = true
	}
	e.recordImports(pkg, f, p, child, fileOf)
	e.renderField(p, num, sb)
	return 1, nil
}
