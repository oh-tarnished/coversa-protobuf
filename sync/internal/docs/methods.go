// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// methods.go writes the resource-name section, the instance axes and the
// method surface.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
)

// resourceNames writes the packet figure and the parsing snippets.
func (g *Generator) resourceNames(pkg *model.Package, sb *strings.Builder) {
	name := sampleName(pkg)

	sb.WriteString("## Resource names\n\n")
	fmt.Fprintf(sb, "%s\n\n", wrap(fmt.Sprintf(
		"Every %s is addressed by a name that alternates collection and identifier, "+
			"per [AIP-122](https://aip.dev/122):", pkg.Singular)))

	sb.WriteString("```mermaid\npacket\n")
	offset := 0
	for i, seg := range strings.Split(name, "/") {
		if i > 0 {
			fmt.Fprintf(sb, "%d: \"/\"\n", offset)
			offset++
		}
		end := offset + len(seg) - 1
		fmt.Fprintf(sb, "%d-%d: %q\n", offset, end, seg)
		offset = end + 1
	}
	sb.WriteString("```\n\n")

	fmt.Fprintf(sb, "%s\n\n", wrap(fmt.Sprintf(
		"%d characters, alternating, which is what [AIP-123](https://aip.dev/123) "+
			"requires. An identifier is 80 random bits in Crockford base32, prefixed with "+
			"the resource's singular so the name opens with a letter — the randomness half "+
			"of a ULID with the timestamp half removed, because a ULID's leading characters "+
			"*are* a timestamp and a resource name should not publish when the record was "+
			"created. The vehicle is the exception: its identifier is its VIN, which is "+
			"already unique and already meaningful, the case AIP-122 prefers.", len(name))))

	g.templateSnippets(pkg, name, sb)
}

// templateSnippets shows the pattern being parsed rather than split by hand.
func (g *Generator) templateSnippets(pkg *model.Package, name string, sb *strings.Builder) {
	sb.WriteString(wrap("Rather than splitting that string by hand, use "+
		"[`resourcename`](https://github.com/the-protobuf-project/resourcename), which "+
		"implements AIP-122 templates in Go, Python, Rust, TypeScript, Swift and C:") +
		"\n\n")

	fmt.Fprintf(sb, "```go\nt := resourcename.ResourceTemplate(%q)\nt.Parse(%q)\n// %s\n```\n\n",
		pkg.Pattern, name, parsedMap(pkg))
	fmt.Fprintf(sb, "```python\nt = resourcename.ResourceTemplate(%q)\nt.parse(%q)\n# %s\n```\n\n",
		pkg.Pattern, name, parsedDict(pkg))

	sb.WriteString(wrap("The template above is the `pattern` this resource declares in "+
		"its `google.api.resource` annotation, so the two cannot drift.") + "\n\n")
}

// parsedMap renders what Go's Parse returns.
func parsedMap(pkg *model.Package) string {
	var parts []string
	for _, p := range chain(pkg) {
		parts = append(parts, p.Singular+":"+sampleID(p))
	}
	return "map[" + strings.Join(parts, " ") + "]"
}

// parsedDict renders what Python's parse returns.
func parsedDict(pkg *model.Package) string {
	var parts []string
	for _, p := range chain(pkg) {
		parts = append(parts, fmt.Sprintf("'%s': '%s'", p.Singular, sampleID(p)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// chain is the naming ancestry, outermost first.
func chain(pkg *model.Package) []*model.Package {
	var out []*model.Package
	for p := pkg; p != nil; p = p.NameParent {
		if p.IsList || p.NameParent == nil {
			out = append([]*model.Package{p}, out...)
		}
	}
	return out
}

// instances draws the axes an instanced branch expands across.
func (g *Generator) instances(pkg *model.Package, sb *strings.Builder) {
	axes := pkg.Root.Instances
	if len(axes) == 0 {
		return
	}

	fmt.Fprintf(sb, "## Which %s\n\n```mermaid\nflowchart LR\n", pkg.Singular)
	if len(axes) == 1 {
		for i, v := range axes[0] {
			fmt.Fprintf(sb, "  A%d[\"%s\"]\n", i, v)
		}
	} else {
		for i, outer := range axes[0] {
			fmt.Fprintf(sb, "  O%d[\"%s\"]\n", i, outer)
			for j, inner := range axes[1] {
				fmt.Fprintf(sb, "  O%d --> I%d_%d[\"%s\"]\n", i, i, j, inner)
			}
		}
	}
	sb.WriteString("```\n\n")

	fmt.Fprintf(sb, "%s\n\n", wrap(fmt.Sprintf(
		"VSS expands this branch across %s, so the combination names one %s — which is "+
			"what makes it a resource rather than a repeated field. The axes are recorded "+
			"in [`codec/manifest.json`](../../../../../codec/manifest.json), which is the "+
			"only thing that says how to map an id back to a VSS path.",
		axisPhrase(axes), pkg.Singular)))
}

// axisPhrase describes the axes in prose.
func axisPhrase(axes [][]string) string {
	parts := make([]string, len(axes))
	for i, axis := range axes {
		parts[i] = strings.Join(axis, ", ")
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " × ")
}
