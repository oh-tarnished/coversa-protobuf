// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// doc.go holds the comment machinery: wrapping, the message documentation
// block, and the unit symbol table field comments read from.

import (
	"strings"

	"github.com/the-protobuf-project/vdm/sync/describe"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/vspec"
)

// docWidth is the column comment text wraps at, leaving room for the `// `
// prefix and an indent inside a message.
const docWidth = 74

// docBlock renders text as a `//` comment block at the given indent.
func docBlock(text, indent string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var out strings.Builder
	for _, para := range strings.Split(text, "\n") {
		if para = strings.TrimSpace(para); para == "" {
			out.WriteString(indent + "//\n")
			continue
		}
		for _, line := range wrap(para, docWidth-len(indent)) {
			out.WriteString(indent + "// " + line + "\n")
		}
	}
	return out.String()
}

// wrap breaks a paragraph into lines of at most width characters, never
// splitting a word.
func wrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := []string{}
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	return append(lines, cur)
}

// trimTrailingBlank removes trailing newlines from a builder and restores a
// single one, so a block closes immediately after its last member.
func trimTrailingBlank(sb *strings.Builder) {
	out := strings.TrimRight(sb.String(), "\n")
	sb.Reset()
	sb.WriteString(out + "\n")
}

// messageDoc builds a message's comment: the source description, what it is
// in VSS terms, and what it is in AIP terms.
func (e *Emitter) messageDoc(pkg *model.Package, t *vspec.Node, root bool) string {
	doc := describe.Message(model.MessageName(t.Name), t)

	notes := []string{"VSS: " + t.FQN + "."}
	if len(t.Instances) > 0 {
		notes = append(notes, "Instanced: "+instanceNote(t)+
			" The axes together name one occurrence, which is why each is a "+
			"resource of its own rather than a repeated field.")
	}
	if root {
		notes = append(notes, resourceNote(pkg))
	}
	if len(notes) > 0 {
		doc += "\n\n" + strings.Join(notes, "\n\n")
	}
	return doc + "\n\nReference: COVESA Vehicle Signal Specification.\n" +
		"https://covesa.github.io/vehicle_signal_specification/"
}

// resourceNote says what kind of resource a package root is.
func resourceNote(pkg *model.Package) string {
	switch {
	case pkg.TopLevel() && pkg.Family == model.FamilyVSS:
		return "The root resource of the vehicle tree, named \"" + pkg.Pattern + "\".\n\n" +
			"It carries only its own signals. Each branch beneath it -- the cabin, " +
			"the powertrain, the chassis -- is a resource of its own in its own " +
			"package, because AIP-215 <https://aip.dev/215> forbids a field naming " +
			"a message in another package. Navigate to one by resource name."
	case pkg.Parent != nil:
		kind := "A child resource of " + model.MessageName(pkg.Parent.Root.Name)
		if pkg.Singleton() {
			kind = "A singleton child resource of " +
				model.MessageName(pkg.Parent.Root.Name) +
				", AIP-156 <https://aip.dev/156>"
		}
		return kind + ", named \"" + pkg.Pattern + "\". Its parent is a declared " +
			"resource in " + pkg.Parent.ProtoPackage() + ", which is what makes " +
			"this pattern legal rather than a reference to a collection nothing " +
			"can create."
	default:
		return "A root resource, named \"" + pkg.Pattern + "\"."
	}
}

// instanceNote spells out the axes a branch expands across.
func instanceNote(n *vspec.Node) string {
	axes := make([]string, 0, len(n.Instances))
	for _, axis := range n.Instances {
		axes = append(axes, strings.Join(axis, ", "))
	}
	return strings.Join(axes, " × ") + "."
}
