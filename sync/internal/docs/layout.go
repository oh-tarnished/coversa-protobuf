// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// layout.go draws where a resource sits: its parent, its own child resources,
// and the embedded messages that travel with it.
//
// The distinction the figure exists to make is the one AIP-215 forces and
// that nothing in the protobuf states plainly: a *child resource* is
// addressed by its own name and fetched from its own service, while an
// *embedded message* has no name and arrives inside this one. Both look like
// nested structure in a schema browser; only one is separately reachable.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// layout writes the "Where it sits" figure.
func (g *Generator) layout(pkg *model.Package, sb *strings.Builder) {
	embedded := embeddedTypes(pkg)
	children := g.childrenOf(pkg)
	if pkg.NameParent == nil && len(embedded) == 0 && len(children) == 0 {
		return
	}

	sb.WriteString("## Where it sits\n\n```mermaid\nflowchart LR\n")

	self := "R"
	if parent := pkg.NameParent; parent != nil {
		fmt.Fprintf(sb, "  P[\"%s<br/><code>%s</code>\"]\n", parent.ResourceName(), parent.Pattern)
	}
	fmt.Fprintf(sb, "  %s[\"%s<br/><code>%s</code>\"]\n", self, pkg.ResourceName(), tail(pkg.Pattern))

	for i, c := range children {
		fmt.Fprintf(sb, "  C%d[\"%s<br/><code>%s</code>\"]\n", i, c.ResourceName(), tail(c.Pattern))
	}
	for i, e := range embedded {
		fmt.Fprintf(sb, "  E%d[\"%s\"]\n", i, model.MessageName(e.Name))
	}
	sb.WriteString("\n")

	if pkg.NameParent != nil {
		fmt.Fprintf(sb, "  P -->|\"%s\"| %s\n", holds(pkg), self)
	}
	for i, c := range children {
		fmt.Fprintf(sb, "  %s -->|\"%s\"| C%d\n", self, holds(c), i)
	}
	for i := range embedded {
		fmt.Fprintf(sb, "  %s --- E%d\n", self, i)
	}

	sb.WriteString("\n  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;\n")
	sb.WriteString("  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;\n")
	sb.WriteString("  class " + resClasses(pkg, len(children)) + " res;\n")
	if len(embedded) > 0 {
		sb.WriteString("  class " + list("E", len(embedded)) + " emb;\n")
	}
	sb.WriteString("```\n\n")

	g.layoutNote(pkg, embedded, sb)
}

// layoutNote explains the two edge kinds the figure draws.
func (g *Generator) layoutNote(pkg *model.Package, embedded []*vspec.Node, sb *strings.Builder) {
	var notes []string

	// A resource whose name skips an ancestor is the most confusing thing on
	// the page, so it is explained where it is visible.
	if p := pkg.Parent; p != nil && pkg.NameParent != nil && p != pkg.NameParent {
		notes = append(notes, fmt.Sprintf(
			"In the specification this hangs beneath **%s**, but its name hangs off "+
				"**%s**. AIP-123 requires a name to alternate collection and identifier, "+
				"and `%s` is a singleton whose name ends in a literal — two literals in a "+
				"row is not a legal name. Nothing is lost: %s has one occurrence, so the "+
				"segment would identify nothing, and the full VSS path survives in the "+
				"branch annotation.",
			p.ResourceName(), pkg.NameParent.ResourceName(), p.Singular, p.Singular))
	}
	if len(embedded) > 0 {
		notes = append(notes, fmt.Sprintf(
			"The %d blue-grey %s **embedded messages**, not resources. They have no name "+
				"of their own and travel with the %s.",
			len(embedded), plural(len(embedded), "box is", "boxes are"), pkg.Singular))
	}
	for _, n := range notes {
		sb.WriteString(wrap(n) + "\n\n")
	}
}

// holds labels a containment edge with how many of the child the parent has.
func holds(child *model.Package) string {
	if child.IsList {
		return "many, each identified"
	}
	return "exactly one"
}

// embeddedTypes are the package's messages other than its resource.
func embeddedTypes(pkg *model.Package) []*vspec.Node {
	if len(pkg.Types) < 2 {
		return nil
	}
	return pkg.Types[1:]
}

// tail is a pattern's last two segments, elided, so a deep name stays legible
// in a box.
func tail(pattern string) string {
	parts := strings.Split(pattern, "/")
	if len(parts) <= 2 {
		return pattern
	}
	return "…/" + strings.Join(parts[len(parts)-2:], "/")
}

// resClasses lists the node ids drawn as resources.
func resClasses(pkg *model.Package, children int) string {
	ids := []string{"R"}
	if pkg.NameParent != nil {
		ids = append(ids, "P")
	}
	for i := 0; i < children; i++ {
		ids = append(ids, fmt.Sprintf("C%d", i))
	}
	return strings.Join(ids, ",")
}

// list renders prefix0,prefix1,… for a class assignment.
func list(prefix string, n int) string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return strings.Join(ids, ",")
}

// plural picks a verb form.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
