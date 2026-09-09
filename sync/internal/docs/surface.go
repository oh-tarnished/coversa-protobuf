// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// surface.go writes the method surface and the enums.
//
// Which methods exist is read from the resource's shape rather than from the
// emitted service.proto: it is the same decision the emitter makes, and
// taking it from the model keeps the two from disagreeing.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/describe"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// methodSet is the standard methods this resource's shape gets.
//
// A singleton has four fewer, and that is not a reduced surface: AIP-156
// defines the complete method set for a singleton, and Create and Delete are
// not in it. Nothing can make a second cabin or remove the only one.
func (g *Generator) methodSet(pkg *model.Package) []string {
	if pkg.Singleton() {
		return []string{"Get", "Update"}
	}
	return []string{"Get", "List", "Create", "Update", "Delete", "Undelete"}
}

// methods writes the method list, the lifecycle figure and the HTTP bindings.
func (g *Generator) methods(pkg *model.Package, sb *strings.Builder) {
	set := g.methodSet(pkg)

	sb.WriteString("## Methods\n\n")
	names := make([]string, len(set))
	for i, m := range set {
		names[i] = "**`" + m + "`**"
	}
	if pkg.Singleton() {
		fmt.Fprintf(sb, "%s\n\n", wrap("A singleton, so "+strings.Join(names, " · ")+
			" only, per [AIP-156](https://aip.dev/156). Nothing can create a second "+
			"or delete the only one — that is the complete method set for a singleton, "+
			"not a reduced one."))
	} else {
		fmt.Fprintf(sb, "A collection, so the full set: %s.\n\n", strings.Join(names, " · "))
	}

	g.lifecycle(pkg, sb)
	g.bindings(pkg, sb)
}

// lifecycle draws the states a resource moves through.
func (g *Generator) lifecycle(pkg *model.Package, sb *strings.Builder) {
	sb.WriteString("```mermaid\nstateDiagram-v2\n")
	if pkg.Singleton() {
		sb.WriteString("  [*] --> Live: exists with its parent\n")
		sb.WriteString("  Live --> Live: Update · only actuators\n```\n\n")
		return
	}
	sb.WriteString("  [*] --> Live: Create\n")
	sb.WriteString("  Live --> Live: Update · only actuators\n")
	sb.WriteString("  Live --> Deleted: Delete · soft\n")
	sb.WriteString("  Deleted --> Live: Undelete\n")
	sb.WriteString("  Deleted --> [*]: expire_time passes\n```\n\n")
}

// bindings writes the HTTP rules and the note on soft delete.
func (g *Generator) bindings(pkg *model.Package, sb *strings.Builder) {
	sb.WriteString("```http\n")
	fmt.Fprintf(sb, "GET    /v1/{name=%s}\n", pattern(pkg.Pattern))
	fmt.Fprintf(sb, "PATCH  /v1/{%s.name=%s}\n", pkg.Singular, pattern(pkg.Pattern))
	if !pkg.Singleton() {
		fmt.Fprintf(sb, "GET    %s\n", pkg.CollectionPath())
		fmt.Fprintf(sb, "POST   %s\n", pkg.CollectionPath())
		fmt.Fprintf(sb, "DELETE /v1/{name=%s}\n", pattern(pkg.Pattern))
		fmt.Fprintf(sb, "POST   /v1/{name=%s}:undelete\n", pattern(pkg.Pattern))
	}
	sb.WriteString("```\n\n")

	if !pkg.Singleton() {
		fmt.Fprintf(sb, "%s\n\n", wrap(fmt.Sprintf(
			"A deleted %s is still returned by `Get` and hidden from `List` unless "+
				"`show_deleted` is set — which is what makes `Undelete` meaningful.",
			pkg.Singular)))
	}
}

// pattern turns a resource name pattern into an HTTP template body.
func pattern(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if strings.HasPrefix(s, "{") {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, "/")
}

// enums writes every enum the package declares.
func (g *Generator) enums(pkg *model.Package, sb *strings.Builder) {
	list := pkg.Enums()
	if len(list) == 0 {
		return
	}
	sb.WriteString("## Enums\n\n")

	for _, e := range list {
		name := g.M.EnumName(e.FQN)
		fmt.Fprintf(sb, "#### `%s`\n\n", name)
		if doc := firstParagraph(describe.Enum(name, e)); doc != "" {
			fmt.Fprintf(sb, "%s\n\n", wrap(doc))
		}

		prefix := naming.Screaming(name)
		values := plan.EnumValues(e)
		bare := make([]string, 0, len(values)+1)
		if len(values) == 0 || values[0].Number != 0 {
			bare = append(bare, "`UNSPECIFIED`")
		}
		for _, v := range values {
			bare = append(bare, "`"+naming.Screaming(v.Name)+"`")
		}
		fmt.Fprintf(sb, "%s\n", wrap(strings.Join(bare, " · ")))
		fmt.Fprintf(sb, "<sub>emitted as `%s`, and so on</sub>\n\n",
			plan.EnumValueName(prefix, firstValue(values)))
	}
}

// firstValue is the name of an enum's first declared value.
func firstValue(values []vspec.EnumEntry) string {
	if len(values) == 0 {
		return "UNSPECIFIED"
	}
	return values[0].Name
}
