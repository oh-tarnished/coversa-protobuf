// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package emit

// vocab.go emits protobuf/covesa/vss/annotations/v1: the annotation
// vocabulary every generated VSS package imports.
//
// Emitted rather than hand-maintained because two of its four files are
// transcriptions of the source model -- Unit and QuantityKind are the unit
// enums flattened -- and a transcription that drifts from its source is worse
// than no transcription. The other two are stable, and are here so that
// nothing under protobuf/ is maintained by hand.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/the-protobuf-project/vdm/sync/internal/model"
	"github.com/the-protobuf-project/vdm/sync/internal/sdl"
)

// quantity is one quantity kind and the units belonging to it.
type quantity struct {
	kind  string   // the source model's name, e.g. "angular-speed"
	enum  string   // the source enum, e.g. AngularSpeedUnitEnum
	units []string // its unit values, in declaration order
}

// generateVocab writes the four vocabulary files.
func (e *Emitter) generateVocab(defs []sdl.Def, out string) (int, error) {
	dir := filepath.Join(out, "vss", model.VocabName, "v1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	qs := quantities(defs)
	files := map[string]string{
		"unit.proto":          e.renderUnits(qs),
		"quantity_kind.proto": e.renderQuantityKinds(qs),
		// The two static bodies are stored without a trailing newline, which
		// a Go raw string would otherwise make invisible in the source; it is
		// restored here so every emitted file ends the same way.
		"options.proto":     e.vocabHeader("OptionsProto", optionsImports) + optionsBody + "\n",
		"annotations.proto": e.vocabHeader("AnnotationsProto", annotationsImports) + annotationsBody + "\n",
	}
	for base, body := range files {
		if err := writeFile(dir, base, body); err != nil {
			return 0, err
		}
	}

	// Field comments read unit symbols from the same table the enum is
	// rendered from, rather than a second copy that could disagree.
	for _, q := range qs {
		for _, u := range q.units {
			unitSymbols["UNIT_"+normaliseUnit(u)] = unitSymbolTable[u]
		}
	}
	return len(files), nil
}

// quantities extracts the unit vocabulary from the parsed specification.
func quantities(defs []sdl.Def) []quantity {
	var out []quantity
	for _, d := range defs {
		if d.Kind != "enum" || !model.IsUnitEnum(d.Name) {
			continue
		}
		q := quantity{enum: d.Name, kind: quotedName(d.Doc)}
		for _, v := range d.Values {
			q.units = append(q.units, v.Name)
		}
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].enum < out[j].enum })
	return out
}

// quotedName pulls the quantity kind out of a description like
// `Units for "angular-speed"`, falling back to the whole description.
func quotedName(doc string) string {
	open := strings.Index(doc, "\"")
	if open < 0 {
		return doc
	}
	close := strings.Index(doc[open+1:], "\"")
	if close < 0 {
		return doc
	}
	return doc[open+1 : open+1+close]
}

// normaliseUnit maps a source unit spelling onto this schema's Unit enum.
func normaliseUnit(v string) string {
	if v == "U_N_I_X_TIMESTAMP" {
		return "UNIX_TIMESTAMP"
	}
	return v
}

// renderUnits writes unit.proto.
func (e *Emitter) renderUnits(qs []quantity) string {
	var sb strings.Builder
	sb.WriteString(e.vocabHeader("UnitProto", nil))
	sb.WriteString(docBlock("Unit is every unit of measurement the source model declares, "+
		"flattened into one enum.\n\n"+
		"Flattened because a protobuf option field has one type: SignalOptions.unit "+
		"must name a single enum, where the source model has one per quantity kind. "+
		"The kind is not lost -- it moves to SignalOptions.quantity_kind, and "+
		"QuantityKind names the same groups.\n\n"+
		"The symbol in each comment is the unit as VSS writes it.\n\n"+
		"Reference: COVESA VSS unit catalogue.\n"+
		"https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/units.yaml", ""))
	sb.WriteString("enum Unit {\n  // Not specified.\n  UNIT_UNSPECIFIED = 0;\n")

	n := 0
	for _, q := range qs {
		sb.WriteString("\n  // " + q.kind + "\n")
		for _, u := range q.units {
			n++
			doc := unitSymbolTable[u]
			if doc == "" {
				doc = strings.ToLower(strings.ReplaceAll(u, "_", " "))
			}
			doc += "."
			if note, ok := unitNotes[u]; ok {
				doc += " " + note
			}
			sb.WriteString(docBlock(doc, "  "))
			fmt.Fprintf(&sb, "  UNIT_%s = %d;\n", normaliseUnit(u), n)
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

// renderQuantityKinds writes quantity_kind.proto.
func (e *Emitter) renderQuantityKinds(qs []quantity) string {
	var sb strings.Builder
	sb.WriteString(e.vocabHeader("QuantityKindProto", nil))
	sb.WriteString(docBlock("QuantityKind is the physical quantity a Unit measures.\n\n"+
		"These are the groups the source model declares as separate unit enums. "+
		"Unit flattens them into one; this names the group a given unit belongs "+
		"to, so a generator can tell which conversions are meaningful.\n\n"+
		"Reference: COVESA VSS quantity catalogue.\n"+
		"https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/quantities.yaml", ""))
	sb.WriteString("enum QuantityKind {\n  // Not specified.\n  QUANTITY_KIND_UNSPECIFIED = 0;\n\n")

	for i, q := range qs {
		var symbols []string
		for _, u := range q.units {
			if s := unitSymbolTable[u]; s != "" {
				symbols = append(symbols, s)
			}
		}
		sb.WriteString(docBlock("\""+q.kind+"\": "+strings.Join(symbols, ", ")+".", "  "))
		fmt.Fprintf(&sb, "  QUANTITY_KIND_%s = %d;\n\n",
			strings.ToUpper(strings.ReplaceAll(q.kind, "-", "_")), i+1)
	}
	trimTrailingBlank(&sb)
	sb.WriteString("}\n")
	return sb.String()
}

// vocabHeader renders the licence and file options for a vocabulary file.
func (e *Emitter) vocabHeader(class string, imports []string) string {
	var sb strings.Builder
	sb.WriteString(e.M.Spec.Banner())
	sb.WriteString("syntax = \"proto3\";\n\npackage " + model.VocabPackage + ";\n\n")
	for _, i := range imports {
		sb.WriteString("import \"" + i + "\";\n")
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("option java_multiple_files = true;\n")
	fmt.Fprintf(&sb, "option java_outer_classname = %q;\n", class)
	sb.WriteString("option java_package = \"" + javaVocabPackage + "\";\n\n")
	return sb.String()
}

// javaVocabPackage is the vocabulary's Java package, per AIP-191.
const javaVocabPackage = "io.github.theprotobufproject.protobuf.covesa.vss." +
	model.VocabName + ".v1"

// optionsImports and annotationsImports are what the two static files need.
var (
	optionsImports = []string{
		model.VocabRoot + "/quantity_kind.proto",
		model.VocabRoot + "/unit.proto",
	}
	annotationsImports = []string{
		"google/protobuf/descriptor.proto",
		model.VocabRoot + "/options.proto",
	}
)
