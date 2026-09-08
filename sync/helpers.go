// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// helpers.go holds the small rendering utilities: comment wrapping, enum
// output, the resource-name pattern regex, and the unit symbol table used in
// field comments.

import (
	"fmt"
	"strconv"
	"strings"
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
		para = strings.TrimSpace(para)
		if para == "" {
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
	var lines []string
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

// messageDoc builds the documentation block for a message, combining the
// source description with what the message is in AIP terms.
func (m *Model) messageDoc(pkg *Package, t *Def, root bool) string {
	doc := t.Doc
	if doc == "" {
		doc = messageName(t.Name) + " is a node of the COVESA Vehicle Signal Specification."
	}
	var notes []string
	if v, ok := t.Directive("vspec"); ok {
		if fqn, ok := v.Arg("fqn"); ok {
			notes = append(notes, "VSS: "+fqn+".")
		}
	}
	if _, isTag := t.Directive("instanceTag"); isTag {
		notes = append(notes, "An S2DM instance tag: it says which member of a repeated branch a "+
			"value belongs to, and carries no signal of its own.")
	}
	if root && pkg.TopLevel() && pkg.Family == FamilyVSS {
		notes = append(notes, "The root resource of the vehicle tree, named \""+pkg.Pattern+"\".\n\n"+
			"It carries only its own signals. Each branch beneath it -- the cabin, "+
			"the powertrain, the chassis -- is a resource of its own in its own "+
			"package, because AIP-215 <https://aip.dev/215> forbids a field naming "+
			"a message in another package. Navigate to one by resource name.")
	} else if root && pkg.Parent != nil {
		pn := messageName(pkg.Parent.Root.Name)
		kind := "A child resource of " + pn
		if pkg.Singleton() {
			kind = "A singleton child resource of " + pn + ", AIP-156 <https://aip.dev/156>"
		}
		notes = append(notes, kind+", named \""+pkg.Pattern+"\". "+
			"Its parent is a declared resource in "+protoPackage(pkg.Parent)+", "+
			"which is what makes this pattern legal rather than a reference to a "+
			"collection nothing can create.")
	} else if root {
		notes = append(notes, "A root resource, named \""+pkg.Pattern+"\".")
	}
	if len(notes) > 0 {
		doc += "\n\n" + strings.Join(notes, "\n\n")
	}
	doc += "\n\nReference: COVESA Vehicle Signal Specification.\n" +
		"https://covesa.github.io/vehicle_signal_specification/"
	return doc
}

// renderEnum writes one enum, its values prefixed with the enum name so they
// do not collide in the package scope proto3 gives them.
func renderEnum(f *File, e *Def, sb *strings.Builder) {
	name := enumName(e.Name)
	prefix := screaming(name)

	doc := e.Doc
	if doc == "" {
		doc = name + " is an allowed-value set from the source model."
	}
	if v, ok := e.Directive("vspec"); ok {
		if fqn, ok := v.Arg("fqn"); ok {
			doc += "\n\nVSS: " + fqn + "."
		}
	}
	sb.WriteString(docBlock(doc, ""))
	sb.WriteString("enum " + name + " {\n")

	// AIP-126 requires a zero value meaning "unspecified". Where the source
	// model already opens with UNDEFINED, that value *is* the zero rather than
	// a second spelling of it.
	num := 0
	if len(e.Values) == 0 || e.Values[0].Name != "UNDEFINED" {
		sb.WriteString("  // Not specified.\n")
		sb.WriteString("  " + prefix + "_UNSPECIFIED = 0;\n\n")
		num = 1
	}
	for i, v := range e.Values {
		vdoc := v.Doc
		src, hasSrc := "", false
		if d, ok := v.Directive("vspec"); ok {
			if s, ok := d.Arg("originalName"); ok {
				src, hasSrc = s, true
			}
		}
		if vdoc == "" {
			if i == 0 && v.Name == "UNDEFINED" {
				vdoc = "Not specified."
			} else {
				vdoc = humanise(v.Name) + "."
			}
		}
		if hasSrc {
			vdoc += "\n\nThe source model spells this \"" + src + "\"."
		}
		sb.WriteString(docBlock(vdoc, "  "))
		val := enumValueName(prefix, v.Name)
		if i == 0 && v.Name == "UNDEFINED" {
			val = prefix + "_UNSPECIFIED"
		}
		if hasSrc {
			f.Imports[vocabRoot+"/annotations.proto"] = true
			sb.WriteString(fmt.Sprintf("  %s = %d [(%s.allowed_value) = {source_spelling: %q}];\n\n", val, num, vocabPackage, src))
		} else {
			sb.WriteString(fmt.Sprintf("  %s = %d;\n\n", val, num))
		}
		num++
	}
	// Trim the blank line after the last value: buf format would remove it,
	// and emitting output the formatter has to correct makes `just gen`
	// produce a diff against its own previous run.
	out := strings.TrimRight(sb.String(), "\n")
	sb.Reset()
	sb.WriteString(out + "\n}\n")
}

// humanise turns an enum value name into a sentence fragment for a comment.
func humanise(v string) string {
	words := splitWords(v)
	if len(words) == 0 {
		return v
	}
	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	return strings.Join(words, " ")
}

// patternToRegex turns a resource name pattern into the anchored regex that
// validates it: every `{segment}` becomes one path component.
func patternToRegex(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "{") {
			parts[i] = "[^/]+"
		}
	}
	return "^" + strings.Join(parts, "/") + "$"
}

// unitSymbols maps a Unit enum value onto the symbol VSS writes, for comments.
var unitSymbols = map[string]string{}

// unitSymbol renders a Unit enum value as its written symbol, falling back to
// a readable form of the constant when the table has no entry.
func unitSymbol(u string) string {
	if s, ok := unitSymbols[u]; ok {
		return s
	}
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(u, "UNIT_"), "_", " "))
}

// itoa renders a field number.
func itoa(n int) string { return strconv.Itoa(n) }
