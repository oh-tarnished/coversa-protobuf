// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// diagram.go renders the reference's Mermaid figures.
//
// Generated from the model rather than drawn, for the reason the tables are:
// a figure that states "18 singletons" stops being true the moment the
// specification moves, and nothing reports it -- the picture still renders,
// it just lies. Here the counts are counted.
//
// They are written straight into the README rather than into standalone .mmd
// files. GitHub renders a fenced ```mermaid block natively, and a diagram
// file nothing includes is a diagram nobody reads.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
)

// erMax is how many children of one parent the relationship diagram draws
// before collapsing the rest into a counted edge.
//
// Vehicle has 38. Drawing them all produces a figure no one can read; drawing
// none hides the shape. The cut is arbitrary, the collapse is not -- the
// remainder is counted, so the picture is honest about what it omits.
const erMax = 7

// relationships renders the entity-relationship figure: every resource and
// how it relates to the others.
func (g *Generator) relationships(sb *strings.Builder) {
	sb.WriteString("## How the resources relate\n\n")
	sb.WriteString("```mermaid\nerDiagram\n")

	for _, root := range g.roots() {
		g.erChildren(root, sb)
	}
	g.erAssociations(sb)
	sb.WriteString("```\n\n")

	singles, collections := g.shapeCounts()
	fmt.Fprintf(sb, "> **Two kinds of edge, and they are not interchangeable.** A `||--||` or\n"+
		"> `||--o{` is *containment*: the child's resource name extends the parent's,\n"+
		"> so a seat is `vehicles/{vehicle}/seats/{seat}` and cannot exist without its\n"+
		"> vehicle. A `}o--||` is an *association*: the field holds a `string` carrying\n"+
		"> the other resource's name, because AIP-215 forbids naming a message in\n"+
		"> another proto package.\n>\n"+
		"> %d singletons and %d collections across %d resources.\n\n",
		singles, collections, len(g.M.Packages))
}

// erChildren writes one parent's containment edges, then recurses.
func (g *Generator) erChildren(parent *model.Package, sb *strings.Builder) {
	children := g.childrenOf(parent)

	var singles, many []*model.Package
	for _, c := range children {
		if c.Singleton() {
			singles = append(singles, c)
			continue
		}
		many = append(many, c)
	}
	g.erRun(parent, singles, "||--||", sb)
	g.erRun(parent, many, "||--o{", sb)

	for _, c := range children {
		g.erChildren(c, sb)
	}
}

// erRun writes one cardinality's edges, collapsing the tail into a count.
func (g *Generator) erRun(parent *model.Package, children []*model.Package, edge string, sb *strings.Builder) {
	if len(children) == 0 {
		return
	}

	shown := children
	if len(children) > erMax {
		shown = children[:erMax]
	}
	for _, c := range shown {
		fmt.Fprintf(sb, "  %s %s %s : %q\n", erName(parent), edge, erName(c), segment(c))
	}
	if rest := len(children) - len(shown); rest > 0 {
		kind := "OTHER_COLLECTIONS"
		if children[0].Singleton() {
			kind = "OTHER_SINGLETONS"
		}
		fmt.Fprintf(sb, "  %s %s %s : \"+%d more\"\n", erName(parent), edge, kind, rest)
	}
}

// erAssociations writes the cross-package references, which are resource
// names rather than embedded messages.
func (g *Generator) erAssociations(sb *strings.Builder) {
	type edge struct{ from, to, label string }
	var edges []edge

	for _, pkg := range g.M.Packages {
		for _, c := range pkg.Root.Children {
			if c.Ref == "" {
				continue
			}
			other := g.M.PackageOf(c.Ref)
			if other == nil {
				continue
			}
			edges = append(edges, edge{erName(pkg), erName(other), c.Name})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from != edges[j].from {
			return edges[i].from < edges[j].from
		}
		return edges[i].label < edges[j].label
	})
	for _, e := range edges {
		fmt.Fprintf(sb, "  %s }o--|| %s : %q\n", e.from, e.to, e.label)
	}
}

// erName renders a resource as a Mermaid entity name.
func erName(p *model.Package) string { return strings.ToUpper(p.Name) }

// segment is the name segment a parent's edge adds: a collection contributes
// its plural, a singleton its singular.
func segment(p *model.Package) string {
	if p.Singleton() {
		return p.Singular
	}
	return p.Plural
}

// roots returns the packages with no parent, sorted by name.
func (g *Generator) roots() []*model.Package {
	var out []*model.Package
	for _, p := range g.M.Packages {
		if p.TopLevel() {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// childrenOf returns the packages whose *name* hangs beneath parent.
func (g *Generator) childrenOf(parent *model.Package) []*model.Package {
	var out []*model.Package
	for _, p := range g.M.Packages {
		if p.NameParent == parent {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ResourceName() < out[j].ResourceName() })
	return out
}

// shapeCounts totals the resources by shape.
func (g *Generator) shapeCounts() (singletons, collections int) {
	for _, p := range g.M.Packages {
		if p.Singleton() {
			singletons++
			continue
		}
		collections++
	}
	return singletons, collections
}
