// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// signals.go writes the field reference, split by whether the vehicle will
// accept a write.
//
// That split is the organising fact of this schema and it is invisible in a
// single alphabetical table. VSS marks each signal an attribute, a sensor or
// an actuator; only an actuator is writable, and an `update_mask` naming
// either of the others is rejected. A reader deciding what to send needs the
// writable ones together, not interleaved with 30 they cannot touch.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/describe"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// aipFieldCount is the identity and lifecycle fields every resource carries:
// name, uid, etag and the four timestamps.
const aipFieldCount = 7

// signals writes the two field sections and the collapsed AIP table.
func (g *Generator) signals(pkg *model.Package, sb *strings.Builder) {
	writable, readonly := g.signalSplit(pkg)
	if len(writable)+len(readonly) == 0 {
		return
	}

	sb.WriteString("## Signals\n\n")
	fmt.Fprintf(sb, "%d fields: %d AIP identity and lifecycle, %d VSS signals. Field "+
		"numbers 8–15 are reserved for identity fields a later revision may add, so "+
		"adding one never renumbers a signal.\n\n",
		aipFieldCount+len(writable)+len(readonly), aipFieldCount, len(writable)+len(readonly))

	if len(writable) > 0 {
		sb.WriteString("### Writable — actuators\n\n")
		sb.WriteString("The vehicle accepts these in an `update_mask`.\n\n")
		g.signalTable(writable, sb)
	}
	if len(readonly) > 0 {
		sb.WriteString("### Read-only — sensors and attributes\n\n")
		sb.WriteString(wrap("The vehicle reports these. An `update_mask` naming one is "+
			"**rejected**, not silently ignored.") + "\n\n")
		g.signalTable(readonly, sb)
	}
	g.aipTable(pkg, sb)
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

// signalSplit divides a resource's own signals into writable and read-only,
// each in specification order.
//
// Only the resource message's fields: an embedded message's signals are
// documented with the message, and hoisting them here would list a field
// under a name no request can address.
func (g *Generator) signalSplit(pkg *model.Package) (writable, readonly []exampleSignal) {
	for _, c := range pkg.Root.Children {
		if c.Kind == vspec.KindBranch && c.Ref == "" && !pkg.Holds(c.FQN) {
			continue
		}
		p, err := g.Planner.Field(pkg.Root, c, true)
		if err != nil {
			continue
		}
		s := exampleSignal{field: p, node: c}
		if contains(p.Behavior, "OUTPUT_ONLY") {
			readonly = append(readonly, s)
			continue
		}
		writable = append(writable, s)
	}
	sortSignals(writable)
	sortSignals(readonly)
	return writable, readonly
}

// sortSignals orders a group by name, so a reader can find a field without
// knowing where the specification happened to declare it.
func sortSignals(group []exampleSignal) {
	sort.SliceStable(group, func(i, j int) bool {
		return group[i].field.Name < group[j].field.Name
	})
}

// contains reports whether a behaviour list names b.
func contains(list []string, b string) bool {
	for _, v := range list {
		if v == b {
			return true
		}
	}
	return false
}

// enumOf is the emitted enum name for a node, or "".
func (g *Generator) enumOf(n *vspec.Node) string {
	if !model.HasEnum(n) {
		return ""
	}
	return g.M.EnumName(n.FQN)
}

// values renders a node's enum values.
func (g *Generator) values(n *vspec.Node) []vspec.EnumEntry { return plan.EnumValues(n) }
