// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// render.go writes the protobuf text for one file.
//
// Output is deliberately not formatted to buf's exact taste here; `buf format
// -w` is run over the tree afterwards and is the authority, so this code
// optimises for being obviously correct rather than for matching a formatter
// byte for byte.

import (
	"sort"
	"strings"
)

// firstVSSFieldNumber leaves 8-15 free below it for AIP identity fields a
// future revision may add, so adding one never renumbers a signal.
const firstVSSFieldNumber = 16

// renderFile produces the complete text of one file.
func (m *Model) renderFile(pkg *Package, f *File, fileOf map[string]string) (string, error) {
	inPkg := map[string]bool{}
	for _, t := range pkg.Types {
		inPkg[t.Name] = true
	}

	var body strings.Builder
	if !f.TypesOnly {
		if err := m.renderMessage(pkg, f, &body, inPkg, fileOf); err != nil {
			return "", err
		}
	}
	for _, e := range f.Enums {
		renderEnum(f, e, &body)
	}
	if f.EnumFile != "" {
		f.Imports[pkg.ImportPath(f.EnumFile)] = true
	}

	var head strings.Builder
	head.WriteString(m.Spec.banner())
	head.WriteString("syntax = \"proto3\";\n\n")
	head.WriteString("package " + protoPackage(pkg) + ";\n\n")

	var imps []string
	for i := range f.Imports {
		imps = append(imps, i)
	}
	sort.Strings(imps)
	for _, i := range imps {
		head.WriteString("import \"" + i + "\";\n")
	}
	if len(imps) > 0 {
		head.WriteString("\n")
	}
	head.WriteString("option java_multiple_files = true;\n")
	head.WriteString("option java_outer_classname = \"" + outerClassname(f.Base) + "\";\n")
	head.WriteString("option java_package = \"" + pkg.JavaPackage() + "\";\n\n")
	return head.String() + body.String(), nil
}

// renderMessage writes one message, as a resource when it is the package root.
func (m *Model) renderMessage(pkg *Package, f *File, sb *strings.Builder, inPkg map[string]bool, fileOf map[string]string) error {
	t := f.Type
	f.Imports["google/api/field_behavior.proto"] = true

	sb.WriteString(docBlock(m.messageDoc(pkg, t, f.Root), ""))
	sb.WriteString("message " + messageName(t.Name) + " {\n")

	if f.Root {
		f.Imports["google/api/resource.proto"] = true
		f.Imports["google/api/field_info.proto"] = true
		f.Imports["buf/validate/validate.proto"] = true
		f.Imports["google/protobuf/timestamp.proto"] = true
		renderResourceHeader(pkg, sb)
	}
	if v, ok := t.Directive("vspec"); ok {
		el, _ := v.Arg("element")
		fqn, _ := v.Arg("fqn")
		if el != "" {
			f.Imports[vocabRoot+"/annotations.proto"] = true
			sb.WriteString("  option (" + vocabPackage + ".branch) = {\n")
			sb.WriteString("    element: ELEMENT_" + el + "\n")
			if fqn != "" {
				sb.WriteString("    fqn: \"" + fqn + "\"\n")
			}
			sb.WriteString("  };\n\n")
		}
	}

	num := 1
	if f.Root {
		num = firstVSSFieldNumber
	}
	for _, fl := range t.Fields {
		// A field whose type lives in another package cannot be named
		// directly: AIP-215 <https://aip.dev/215> forbids it. There are two
		// cases and they resolve differently.
		if _, isObj := m.Types[fl.Type.Name]; isObj && !inPkg[fl.Type.Name] {
			other := m.packageOf(fl.Type.Name)
			// A child of this resource is reachable by a name derived from
			// this one -- vehicles/{v}/cabin -- so restating it as a field
			// would be redundant and could disagree with the real name.
			if other == nil || other.Parent == pkg {
				continue
			}
			// Anything else is a genuine association the reader cannot
			// derive: which vehicle charged, which person was billed. It
			// becomes the resource name, per AIP-122 <https://aip.dev/122>.
			f.Imports["google/api/resource.proto"] = true
			renderReference(m, t, fl, other, num, sb)
			num++
			continue
		}
		p, err := m.planField(t, fl, f.Root)
		if err != nil {
			return err
		}
		for _, i := range p.Imports {
			f.Imports[i] = true
		}
		if len(p.Validate) > 0 {
			f.Imports["buf/validate/validate.proto"] = true
		}
		if p.Element != "" {
			f.Imports[vocabRoot+"/annotations.proto"] = true
		}
		// The import usually follows the source field's type name, but a
		// field promoted to another message -- a date string becoming a Date
		// -- names a type the source field never mentioned. Resolve on the
		// planned type, falling back to the source name.
		dep, ok := fileOf[p.Type]
		if !ok {
			dep, ok = fileOf[fl.Type.Name]
		}
		if ok && dep != f.Base {
			f.Imports[pkg.ImportPath(dep)] = true
		}
		renderField(p, num, sb)
		num++
	}
	// Trim the blank line after the last field, for the reason renderEnum
	// gives: emitting output buf format has to correct makes `just sync`
	// produce a diff against its own previous run.
	trimTrailingBlank(sb)
	sb.WriteString("}\n")
	return nil
}

// trimTrailingBlank removes trailing newlines from a builder and restores a
// single one, so a block closes immediately after its last member.
func trimTrailingBlank(sb *strings.Builder) {
	out := strings.TrimRight(sb.String(), "\n")
	sb.Reset()
	sb.WriteString(out + "\n")
}
