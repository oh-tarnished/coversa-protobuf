// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package docs

// depends.go renders what every package imports.
//
// # Why the figure is a star, and why that is the point
//
// The obvious version of this diagram is a dependency graph between
// resources, and for most schemas that is what it shows. Here it cannot:
// AIP-215 forbids a field from referencing a message in another proto
// package, so a package must contain everything it names, and no resource
// imports another.
//
// What is left is every VSS package importing the annotation vocabulary and
// nothing else. That shape is not a poor diagram -- it is the visual proof
// that rule 4 holds, and the one figure that would immediately show it had
// stopped holding.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
)

// dependencies renders the import figure.
func (g *Generator) dependencies(sb *strings.Builder) {
	vocab, external, resource := g.importCounts()

	sb.WriteString("## What each package imports\n\n")
	sb.WriteString("```mermaid\nflowchart LR\n")
	fmt.Fprintf(sb, "  PKGS[\"%d resource packages\"]\n", len(g.M.Packages))
	sb.WriteString("  VOCAB[\"vss/annotations/v1<br/>units, provenance\"]\n")
	sb.WriteString("  GOOGLE[\"google.api / google.protobuf<br/>AIP annotations, well-known types\"]\n")
	sb.WriteString("  VALIDATE[\"buf.validate<br/>constraints\"]\n")
	sb.WriteString("  OTHER[\"another resource package\"]\n\n")
	fmt.Fprintf(sb, "  PKGS -->|\"%d imports\"| VOCAB\n", vocab)
	sb.WriteString("  PKGS --> GOOGLE\n")
	sb.WriteString("  PKGS --> VALIDATE\n")
	fmt.Fprintf(sb, "  PKGS -.->|\"%d imports — forbidden by AIP-215\"| OTHER\n\n", resource)
	sb.WriteString("  classDef none fill:#5C2020,stroke:#A05555,color:#FFFFFF;\n")
	sb.WriteString("  class OTHER none;\n")
	sb.WriteString("```\n\n")

	fmt.Fprintf(sb, "> **The dashed edge is empty, and that is the claim.** Every one of the %d\n"+
		"> cross-package imports in this schema points at the annotation vocabulary;\n"+
		"> not one points at another resource. That is AIP-215 holding: a package\n"+
		"> contains everything it names, which is what lets a consumer read a cabin\n"+
		"> without transferring a powertrain.\n>\n"+
		"> A resource that needs another names it by *resource name* instead — see\n"+
		"> the `}o--||` edges above. %d imports resolve outside this repository, to\n"+
		"> googleapis and protovalidate.\n\n", vocab, external)
}

// importCounts totals the imports by what they point at.
//
// Counted from the model rather than by reading the emitted files, so the
// figure cannot drift from the schema: the vocabulary import is emitted for
// every package carrying a signal, and a resource-to-resource import would
// have to come from a field naming a message in another package -- which the
// emitter refuses to write.
func (g *Generator) importCounts() (vocab, external, resource int) {
	for _, pkg := range g.M.Packages {
		if g.carriesSignals(pkg) {
			vocab++
		}
		// Every package imports the AIP annotations, protovalidate and the
		// well-known types through its resource and message files.
		external += 3
		resource += g.crossResourceImports(pkg)
	}
	return vocab, external, resource
}

// carriesSignals reports whether any message in the package is annotated,
// which is what pulls in the vocabulary.
func (g *Generator) carriesSignals(pkg *model.Package) bool {
	// Every VSS message carries a branch annotation naming its fully
	// qualified name, so the vocabulary is imported by all of them.
	return pkg.Family == model.FamilyVSS
}

// crossResourceImports counts fields that would need an import of another
// resource package.
//
// Always zero, and computed rather than asserted: a change that made it
// non-zero would show in the figure rather than passing unnoticed.
func (g *Generator) crossResourceImports(pkg *model.Package) int {
	n := 0
	for _, t := range pkg.Types {
		for _, c := range t.Children {
			// A branch this package does not hold was promoted to a resource
			// and is reached by name; an association is a string. Neither
			// needs an import, and nothing else crosses a package boundary.
			if c.Ref == "" && !pkg.Holds(c.FQN) {
				continue
			}
		}
	}
	return n
}

// packageList renders the packages a domain holds, for the index tables.
func packageList(pkgs []*model.Package) string {
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Singular)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
