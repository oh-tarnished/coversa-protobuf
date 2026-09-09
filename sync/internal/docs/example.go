// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package docs

// example.go writes the worked example: the resource as this API returns it,
// the same resource in the VSS shape, and what happens when a caller tries to
// write a field the vehicle owns.
//
// The two JSON blocks are the point. A consumer arriving at this schema from
// a VSS-native system needs to see what changed -- keys become fully
// qualified names, enum constants drop to the specification's spelling, and
// the AIP fields fall away because VSS declares no signal for them. Saying
// that in prose is not the same as showing it.

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/naming"
	"github.com/oh-tarnished/coversa-protobuf/sync/plan"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// exampleFields is how many signals the JSON blocks show.
//
// Enough to see the shape, few enough to read. Vehicle has 33 signals and a
// complete example would be a wall nobody checks against their own payload.
const exampleFields = 5

// example writes the JSON pair and the update sequence.
func (g *Generator) example(pkg *model.Package, sb *strings.Builder) {
	shown := g.exampleSignals(pkg)
	if len(shown) == 0 {
		return
	}

	fmt.Fprintf(sb, "## Example\n\nA %s as this API returns it:\n\n```json\n{\n", pkg.Singular)
	fmt.Fprintf(sb, "  \"name\": \"%s\",\n", sampleName(pkg))
	sb.WriteString("  \"uid\": \"b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f\",\n")
	rows := make([]string, 0, len(shown))
	for _, s := range shown {
		rows = append(rows, fmt.Sprintf("  %q: %s",
			naming.LowerCamel(s.field.Name), g.sampleValue(s.field, s.node)))
	}
	sb.WriteString(strings.Join(rows, ",\n") + "\n}\n```\n\n")

	g.vssExample(pkg, shown, sb)
	g.updateSequence(pkg, sb)
}

// vssExample writes the same resource keyed by fully qualified VSS name.
func (g *Generator) vssExample(pkg *model.Package, shown []exampleSignal, sb *strings.Builder) {
	sb.WriteString(wrap("The same data in the **VSS shape**, for a peer that speaks the "+
		"specification rather than this schema. `codec.ToVSS` produces it from "+
		"[`codec/manifest.json`](../../../../../codec/manifest.json):") + "\n\n```json\n{\n")

	rows := make([]string, 0, len(shown))
	for _, s := range shown {
		value := g.sampleValue(s.field, s.node)
		// An enum drops to the spelling VSS writes; everything else is the
		// same value under a different key.
		if model.HasEnum(s.node) {
			values := plan.EnumValues(s.node)
			v := values[0]
			if len(values) > 1 {
				v = values[1]
			}
			value = `"` + v.Name + `"`
		}
		rows = append(rows, fmt.Sprintf("  %q: %s", s.node.FQN, value))
	}
	sb.WriteString(strings.Join(rows, ",\n") + "\n}\n```\n\n")

	sb.WriteString(wrap("Note what changed: signals are keyed by **fully qualified VSS "+
		"name**, enum constants drop to the spelling VSS writes, and the AIP fields — "+
		"`name`, `uid` — fall away, because VSS declares no signal for them. "+
		"`codec.FromVSS` reverses all three.") + "\n\n")
}

// updateSequence draws the one request that succeeds and the one that does
// not.
//
// Only drawn where the package has both kinds of signal: the contrast is the
// content, and a figure showing a rejection with nothing to reject against
// teaches nothing.
func (g *Generator) updateSequence(pkg *model.Package, sb *strings.Builder) {
	// Scalars only. An embedded message is not something a caller sets with
	// an update mask, and `{ "airbag": null }` demonstrates nothing.
	writable, readonly := scalars(g.signalSplit(pkg))
	if len(writable) == 0 || len(readonly) == 0 {
		return
	}
	w, r := writable[0], readonly[0]

	fmt.Fprintf(sb, "Writing to a %s, and the one thing that will fail:\n\n", pkg.Singular)
	sb.WriteString("```mermaid\nsequenceDiagram\n  participant C as Client\n")
	fmt.Fprintf(sb, "  participant A as %s\n  participant V as Vehicle\n\n", pkg.ServiceName())

	fmt.Fprintf(sb, "  Note over C,V: %s is an actuator — the API is the source\n", w.field.Name)
	fmt.Fprintf(sb, "  C->>A: PATCH …?updateMask=%s\n", naming.LowerCamel(w.field.Name))
	fmt.Fprintf(sb, "  A->>V: set %s\n  V-->>A: acknowledged\n  A-->>C: 200 OK\n\n", w.field.Name)

	fmt.Fprintf(sb, "  Note over C,V: %s is a %s — the vehicle is the source\n",
		r.field.Name, strings.ToLower(strings.TrimPrefix(r.field.Element, "ELEMENT_")))
	fmt.Fprintf(sb, "  C->>A: PATCH …?updateMask=%s\n", naming.LowerCamel(r.field.Name))
	sb.WriteString("  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field\n```\n\n")

	fmt.Fprintf(sb, "```http\nPATCH /v1/%s?updateMask=%s\n{ %q: %s }\n```\n\n",
		sampleName(pkg), naming.LowerCamel(w.field.Name),
		naming.LowerCamel(w.field.Name), g.sampleValue(w.field, w.node))

	sb.WriteString(wrap(fmt.Sprintf(
		"The rejection is the schema working, not an inconvenience: VSS marks `%s` a %s, "+
			"so the vehicle is the only thing that can set it. A mask naming it fails "+
			"rather than being silently dropped, so a caller that believed it had written "+
			"the value finds out immediately.",
		r.field.Name, strings.ToLower(strings.TrimPrefix(r.field.Element, "ELEMENT_")))) + "\n\n")
}

// scalars drops the fields that hold an embedded message.
func scalars(writable, readonly []exampleSignal) ([]exampleSignal, []exampleSignal) {
	return scalarsOf(writable), scalarsOf(readonly)
}

// scalarsOf keeps the value-carrying fields of one group.
func scalarsOf(group []exampleSignal) []exampleSignal {
	var out []exampleSignal
	for _, s := range group {
		if s.node.Kind == vspec.KindBranch || s.field.Repeated {
			continue
		}
		out = append(out, s)
	}
	return out
}

// exampleSignal pairs a planned field with the node it came from.
type exampleSignal struct {
	field plan.Field
	node  *vspec.Node
}

// exampleSignals picks the scalar signals the JSON blocks show: writable
// first, since those are what a caller acts on.
func (g *Generator) exampleSignals(pkg *model.Package) []exampleSignal {
	writable, readonly := g.signalSplit(pkg)

	var out []exampleSignal
	for _, s := range append(scalarsOf(writable), scalarsOf(readonly)...) {
		out = append(out, s)
		if len(out) == exampleFields {
			break
		}
	}
	return out
}
