// Copyright 2026 Srikanth Kandarpa.
// SPDX-License-Identifier: Apache-2.0

package emit

// vocab.go emits protobuf/covesa/vss/annotations/v1: the annotation
// vocabulary every generated VSS package imports.
//
// The Unit and QuantityKind enums are transcriptions of catalogues VSS ships
// as data — units.yaml and quantities.yaml — so they are rendered from those
// files rather than from a table kept here. A transcription that drifts from
// its source is worse than no transcription.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oh-tarnished/coversa-protobuf/sync/catalog"
	"github.com/oh-tarnished/coversa-protobuf/sync/model"
	"github.com/oh-tarnished/coversa-protobuf/sync/vspec"
)

// generateVocab writes the four vocabulary files.
func (e *Emitter) generateVocab(out string) (int, error) {
	dir := filepath.Join(out, "vss", model.VocabName, "v1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	files := map[string]string{
		"unit.proto":          e.renderUnits(),
		"quantity_kind.proto": e.renderQuantityKinds(),
		"options.proto":       e.vocabHeader("OptionsProto", optionsImports) + optionsBody + "\n",
		"annotations.proto":   e.vocabHeader("AnnotationsProto", annotationsImports) + annotationsBody + "\n",
	}
	for base, body := range files {
		if err := writeFile(dir, base, body); err != nil {
			return 0, err
		}
	}
	return len(files), nil
}

// sortedUnits returns the catalogue's units grouped by quantity, both in a
// stable order, so two runs produce byte-identical output.
func (e *Emitter) sortedUnits() ([]string, map[string][]*vspec.Unit) {
	byQuantity := map[string][]*vspec.Unit{}
	for _, u := range e.M.Units.Units {
		byQuantity[u.Quantity] = append(byQuantity[u.Quantity], u)
	}

	quantities := make([]string, 0, len(byQuantity))
	for q, units := range byQuantity {
		quantities = append(quantities, q)
		sort.Slice(units, func(i, j int) bool { return units[i].Symbol < units[j].Symbol })
	}
	sort.Strings(quantities)
	return quantities, byQuantity
}

// renderUnits writes unit.proto.
func (e *Emitter) renderUnits() string {
	var sb strings.Builder
	sb.WriteString(e.vocabHeader("UnitProto", nil))
	sb.WriteString(docBlock("Unit is every unit of measurement the specification declares, "+
		"flattened into one enum.\n\n"+
		"Flattened because a protobuf option field has one type: SignalOptions.unit "+
		"must name a single enum, where the catalogue groups units by quantity. The "+
		"grouping is not lost -- it moves to SignalOptions.quantity_kind, and "+
		"QuantityKind names the same groups.\n\n"+
		"Transcribed from the catalogue VSS ships, not from a table kept here.\n\n"+
		"Reference: COVESA VSS unit catalogue.\n"+
		"https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/units.yaml", ""))
	sb.WriteString("enum Unit {\n  // Not specified.\n  UNIT_UNSPECIFIED = 0;\n")

	quantities, byQuantity := e.sortedUnits()
	n := 0
	for _, q := range quantities {
		fmt.Fprintf(&sb, "\n  // %s\n", q)
		for _, u := range byQuantity[q] {
			n++
			sb.WriteString(docBlock(unitDoc(u), "  "))
			fmt.Fprintf(&sb, "  %s = %d;\n", catalog.UnitConstant(u.Symbol), n)
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

// unitDoc is the comment for one unit: what VSS writes, what it means, and
// where it sits in the QUDT ontology when the catalogue says.
func unitDoc(u *vspec.Unit) string {
	doc := u.Symbol + " -- " + u.Name + "."
	if u.Definition != "" {
		doc += " " + u.Definition + "."
	}
	if u.Deprecation != "" {
		doc = "Deprecated: " + u.Deprecation + "\n\n" + doc
	}
	if u.QUDT.Unit != "" {
		doc += "\n\nQUDT: " + u.QUDT.Unit
	}
	return doc
}

// renderQuantityKinds writes quantity_kind.proto.
func (e *Emitter) renderQuantityKinds() string {
	var sb strings.Builder
	sb.WriteString(e.vocabHeader("QuantityKindProto", nil))
	sb.WriteString(docBlock("QuantityKind is the physical quantity a Unit measures.\n\n"+
		"These are the groups the catalogue declares. Unit flattens them into one "+
		"enum; this names the group a given unit belongs to, so a generator can tell "+
		"which conversions are meaningful.\n\n"+
		"Reference: COVESA VSS quantity catalogue.\n"+
		"https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/quantities.yaml", ""))
	sb.WriteString("enum QuantityKind {\n  // Not specified.\n  QUANTITY_KIND_UNSPECIFIED = 0;\n\n")

	quantities, byQuantity := e.sortedUnits()
	for i, q := range quantities {
		symbols := make([]string, 0, len(byQuantity[q]))
		for _, u := range byQuantity[q] {
			symbols = append(symbols, u.Symbol)
		}

		doc := "\"" + q + "\": " + strings.Join(symbols, ", ") + "."
		if def := e.M.Units.Quantities[q]; def != nil && def.Definition != "" {
			doc = def.Definition + "\n\nUnits: " + strings.Join(symbols, ", ") + "."
		}
		sb.WriteString(docBlock(doc, "  "))
		fmt.Fprintf(&sb, "  %s = %d;\n\n", catalog.QuantityConstant(q), i+1)
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
const javaVocabPackage = "io.github.ohtarnished.protobuf.covesa.vss." +
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
