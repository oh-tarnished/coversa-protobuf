// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// embedded.go documents the messages that travel inside a resource.
//
// Kept apart from signals.go because it answers a different question. That
// file asks what a caller may write; this one asks what arrives nested, and
// the distinction matters at the call site: an embedded message has no
// resource name, so a field on one is reached through its owner rather than
// addressed on its own.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/describe"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// embedded documents the messages that travel inside the resource.
//
// Collapsed, one per message. They are not separately addressable -- an
// embedded message has no resource name and arrives inside its owner -- so
// listing their fields alongside the resource's own would suggest a caller
// can write `lumbar_height` directly, when the path is
// `backrest.lumbar_height` on the owning resource.
func (g *Generator) embedded(pkg *model.Package, sb *strings.Builder) {
	types := embeddedTypes(pkg)
	if len(types) == 0 {
		return
	}

	sb.WriteString("## Embedded messages\n\n")
	fmt.Fprintf(sb, "%s\n\n", wrap(fmt.Sprintf(
		"%d %s inside the %s and %s no name of their own. Address a field on one "+
			"through its owner — `%s.<field>` — not directly.",
		len(types), plural(len(types), "message travels", "messages travel"),
		pkg.Singular, plural(len(types), "has", "have"),
		naming.Snake(types[0].Name))))

	for _, t := range types {
		fmt.Fprintf(sb, "<details>\n<summary><code>%s</code> — %s</summary>\n\n",
			model.MessageName(t.Name), cell(describe.Message(model.MessageName(t.Name), t)))

		var group []exampleSignal
		for _, c := range t.Children {
			if c.Kind == vspec.KindBranch && c.Ref == "" && !pkg.Holds(c.FQN) {
				continue
			}
			p, err := g.Planner.Field(t, c, false)
			if err != nil {
				continue
			}
			group = append(group, exampleSignal{field: p, node: c})
		}
		if len(group) == 0 {
			sb.WriteString("_No fields of its own._\n\n</details>\n\n")
			continue
		}
		sortSignals(group)
		g.signalTable(group, sb)
		sb.WriteString("</details>\n\n")
	}
}

// signalTable writes one group of signals.
func (g *Generator) signalTable(group []exampleSignal, sb *strings.Builder) {
	sb.WriteString("| Field | Type | Unit | VSS | Description |\n")
	sb.WriteString("| --- | --- | --- | --- | --- |\n")

	for _, s := range group {
		typ := s.field.Type
		if s.field.Repeated {
			typ = "repeated " + typ
		}
		if model.HasEnum(s.node) {
			typ = fmt.Sprintf("[`%s`](#%s)", typ, strings.ToLower(typ))
		} else {
			typ = "`" + typ + "`"
		}

		unit := "—"
		if s.field.Unit != "" {
			unit = "`" + g.symbol(s.field.Unit) + "`"
		}
		fmt.Fprintf(sb, "| `%s` | %s | %s | `%s` | %s |\n",
			s.field.Name, typ, unit, s.node.FQN, cell(describe.Summary(s.field)))
	}
	sb.WriteString("\n")
}

// aipTable writes the identity and lifecycle fields, collapsed.
//
// Every resource carries the same seven, so repeating them at full size on 51
// pages would bury the signals that differ. Collapsed, not omitted: a reader
// who has not seen them before still needs them once.
func (g *Generator) aipTable(pkg *model.Package, sb *strings.Builder) {
	sb.WriteString("<details>\n<summary>Identity and lifecycle — 7 fields every " +
		"resource carries</summary>\n\n")
	sb.WriteString("| Field | Type | Notes |\n| --- | --- | --- |\n")
	fmt.Fprintf(sb, "| `name` | `string` | `%s`, server-assigned |\n", pkg.Pattern)
	sb.WriteString("| `uid` | `string` | server-assigned UUID4, per " +
		"[AIP-148](https://aip.dev/148) |\n")
	sb.WriteString("| `etag` | `string` | pass back on update to make the write " +
		"conditional, per [AIP-154](https://aip.dev/154) |\n")
	sb.WriteString("| `create_time` `update_time` | `Timestamp` | when it was stored here |\n")
	sb.WriteString("| `delete_time` `expire_time` | `Timestamp` | soft delete; " +
		"recoverable until `expire_time` |\n\n</details>\n\n")
}
