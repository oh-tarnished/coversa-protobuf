// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package docs

// render.go writes the three sections of a package README: its service, its
// messages and its enums.

import (
	"fmt"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/describe"
	"github.com/the-protobuf-project/vdm/sync/model"
	"github.com/the-protobuf-project/vdm/sync/naming"
	"github.com/the-protobuf-project/vdm/sync/plan"
	"github.com/the-protobuf-project/vdm/sync/sdl"
)

// method is one row of the service table.
type method struct{ name, request, response, summary string }

// renderService writes the RPC table.
//
// Which methods exist is read from the resource's shape rather than from the
// emitted service.proto: it is the same decision the emitter makes, and
// taking it from the model keeps the two from disagreeing.
func (g *Generator) renderService(pkg *model.Package, sb *strings.Builder) {
	res, plural := pkg.ResourceName(), pkg.PluralType()

	methods := []method{{
		"Get" + res, "Get" + res + "Request", res,
		"Retrieves one " + res + ". `GET " + pkg.HTTPPath() + "`",
	}}
	if !pkg.Singleton() {
		methods = append(methods,
			method{
				"List" + plural, "List" + plural + "Request", "List" + plural + "Response",
				"Lists them, paginated per [AIP-158](https://aip.dev/158). `GET " +
					pkg.CollectionPath() + "`",
			},
			method{
				"Create" + res, "Create" + res + "Request", res,
				"Creates one. `POST " + pkg.CollectionPath() + "`",
			})
	}
	methods = append(methods, method{
		"Update" + res, "Update" + res + "Request", res,
		"Partial update; only an actuator is writable. `PATCH`",
	})
	if !pkg.Singleton() {
		methods = append(methods,
			method{
				"Delete" + res, "Delete" + res + "Request", "google.protobuf.Empty",
				"Soft delete; recoverable until `expire_time`. `DELETE`",
			},
			method{
				"Undelete" + res, "Undelete" + res + "Request", res,
				"Restores a soft-deleted " + res + ". `POST …:undelete`",
			})
	}

	fmt.Fprintf(sb, "## Service `%s`\n\n", pkg.ServiceName())
	sb.WriteString("| Method | Request | Response | Summary |\n")
	sb.WriteString("| --- | --- | --- | --- |\n")
	for _, m := range methods {
		fmt.Fprintf(sb, "| `%s` | `%s` | `%s` | %s |\n",
			m.name, m.request, m.response, m.summary)
	}
	sb.WriteString("\n")
}

// renderMessages writes one table per message in the package.
func (g *Generator) renderMessages(pkg *model.Package, sb *strings.Builder) {
	if len(pkg.Types) == 0 {
		return
	}
	sb.WriteString("## Messages\n\n")

	for _, t := range pkg.Types {
		isRoot := t.Name == pkg.Root.Name
		fmt.Fprintf(sb, "### `%s`\n\n", model.MessageName(t.Name))
		if doc := firstParagraph(describe.Message(model.MessageName(t.Name), t)); doc != "" {
			sb.WriteString(doc + "\n\n")
		}
		if isRoot {
			sb.WriteString("Carries the AIP identity and lifecycle fields — `name`, " +
				"`uid`, `etag` and the four timestamps — ahead of the signals below. " +
				"Field numbers 8–15 are reserved.\n\n")
		}
		g.renderFields(pkg, t, isRoot, sb)
	}
}

// renderFields writes one message's field table.
func (g *Generator) renderFields(pkg *model.Package, t *sdl.Def, isRoot bool, sb *strings.Builder) {
	rows := make([]string, 0, len(t.Fields))
	for _, f := range t.Fields {
		// A field naming a resource in another package is a reference rather
		// than an embedded value; the emitter turns it into a string.
		if _, isObject := g.M.Types[f.Type.Name]; isObject && !pkg.Holds(f.Type.Name) {
			if other := g.M.PackageOf(f.Type.Name); other != nil && other.Parent != pkg {
				rows = append(rows, referenceRow(f, other))
			}
			continue
		}
		p, err := g.Planner.Field(t, f, isRoot)
		if err != nil {
			continue
		}
		rows = append(rows, fieldRow(p))
	}
	if len(rows) == 0 {
		sb.WriteString("_No fields of its own._\n\n")
		return
	}

	sb.WriteString("| Field | Type | Behavior | Unit | Description |\n")
	sb.WriteString("| --- | --- | --- | --- | --- |\n")
	sb.WriteString(strings.Join(rows, "\n") + "\n\n")
}

// fieldRow renders one planned field as a table row.
func fieldRow(p plan.Field) string {
	typ := p.Type
	if p.Repeated {
		typ = "repeated " + typ
	}
	unit := "—"
	if p.Unit != "" {
		unit = "`" + describe.UnitSymbol(strings.TrimPrefix(p.Unit, "UNIT_")) + "`"
	}
	return fmt.Sprintf("| `%s` | `%s` | `%s` | %s | %s |",
		p.Name, typ, strings.Join(p.Behavior, ", "), unit, cell(describe.Summary(p)))
}

// referenceRow renders a cross-package association, which is a resource name.
func referenceRow(f sdl.Field, other *model.Package) string {
	behavior := "OPTIONAL"
	if f.Type.NonNull {
		behavior = "REQUIRED, IMMUTABLE"
	}
	return fmt.Sprintf("| `%s` | `string` | `%s` | — | Resource name of a `%s`, `%s`. |",
		f.Name, behavior, other.ResourceName(), other.Pattern)
}

// renderEnums writes one table per enum in the package.
func (g *Generator) renderEnums(pkg *model.Package, sb *strings.Builder) {
	if len(pkg.Enums) == 0 {
		return
	}
	sb.WriteString("## Enums\n\n")

	for _, e := range pkg.Enums {
		fmt.Fprintf(sb, "### `%s`\n\n", g.M.EnumName(e.Name))
		if doc := firstParagraph(describe.Enum(g.M.EnumName(e.Name), e)); doc != "" {
			sb.WriteString(doc + "\n\n")
		}
		name := g.M.EnumName(e.Name)
		prefix := naming.Screaming(name)

		sb.WriteString("| Value | Description |\n| --- | --- |\n")
		for i, v := range e.Values {
			// The emitted constant, not the source spelling: this table
			// documents the schema, and the schema is what a consumer holds.
			fmt.Fprintf(sb, "| `%s` | %s |\n",
				plan.EnumValueName(prefix, v, i), cell(describe.EnumValue(v, i)))
		}
		sb.WriteString("\n")
	}
}

// cell flattens a doc comment into one Markdown table cell.
//
// A newline would end the row and a pipe would open a column, so both are
// neutralised rather than left to break the table.
func cell(doc string) string {
	doc = firstParagraph(doc)
	if doc == "" {
		return "—"
	}
	return strings.ReplaceAll(doc, "|", "\\|")
}
