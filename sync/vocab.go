// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// vocab.go emits protobuf/covesa/vss/annotations/v1: the vocabulary every
// generated VSS package imports.
//
// Emitted rather than hand-maintained because two of its four files are
// transcriptions of the source model -- Unit and QuantityKind are the thirty
// unit enums flattened -- and a transcription that drifts from its source is
// worse than no transcription. The other two are stable, and are here so that
// nothing under protobuf/ is maintained by hand.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// unitSymbolTable maps a source unit spelling to the symbol VSS writes for it,
// which becomes the comment on the enum value and the text in every field's
// "Unit:" line.
var unitSymbolTable = map[string]string{
	"MILLIMETER": "mm", "CENTIMETER": "cm", "METER": "m", "KILOMETER": "km",
	"INCH": "in", "KILOMETER_PER_HOUR": "km/h", "METERS_PER_SECOND": "m/s",
	"METERS_PER_SECOND_SQUARED": "m/s^2", "CENTIMETERS_PER_SECOND_SQUARED": "cm/s^2",
	"MILLILITER": "ml", "LITER": "l", "CUBIC_CENTIMETERS": "cm^3",
	"DEGREE_CELSIUS": "degC", "DEGREE_FAHRENHEIT": "degF", "DEGREE": "deg",
	"DEGREE_PER_SECOND": "deg/s", "RADIANS_PER_SECOND": "rad/s",
	"WATT": "W", "KILOWATT": "kW", "HORSEPOWER": "PS", "KILOWATT_HOURS": "kWh",
	"GRAM": "g", "KILOGRAM": "kg", "POUND": "lbs", "VOLT": "V",
	"AMPERE": "A", "AMPERE_HOURS": "Ah", "NANOSECOND": "ns", "MILLISECOND": "ms",
	"SECOND": "s", "MINUTE": "min", "HOUR": "h", "DAYS": "day",
	"WEEKS": "weeks", "MONTHS": "months", "YEARS": "years",
	"U_N_I_X_TIMESTAMP": "UNIX timestamp", "ISO_8601": "ISO 8601",
	"MILLIBAR": "mbar", "PASCAL": "Pa", "KILOPASCAL": "kPa",
	"POUNDS_PER_SQUARE_INCH": "psi", "STARS": "stars",
	"GRAMS_PER_SECOND": "g/s", "GRAMS_PER_KILOMETER": "g/km",
	"KILOWATT_HOURS_PER_100_KILOMETERS": "kWh/100km", "WATT_HOUR_PER_KM": "Wh/km",
	"MILLILITER_PER_100_KILOMETERS": "ml/100km", "LITER_PER_100_KILOMETERS": "l/100km",
	"LITER_PER_HOUR": "l/h", "MILES_PER_US_GALLON": "mpg (US)",
	"MILES_PER_IMPERIAL_GALLON":      "mpg (imperial)",
	"MILES_PER_US_GALLON_EQUIVALENT": "MPGe",
	"MILES_PER_US_GALLON_DEPRECATED": "mpg", "KILOMETERS_PER_LITER": "km/l",
	"NEWTON": "N", "KILO_NEWTON": "kN", "NEWTON_METER": "Nm",
	"REVOLUTIONS_PER_MINUTE": "rpm", "HERTZ": "Hz", "CYCLES_PER_MINUTE": "cpm",
	"BEATS_PER_MINUTE": "bpm", "RATIO": "ratio", "PERCENT": "percent",
	"NANO_METER_PER_KILOMETER": "nm/km", "DECIBEL_MILLIWATT": "dBm",
	"DECIBEL": "dB", "OHM": "Ohm", "LUX": "lx",
}

// unitNotes explains a unit whose source spelling needs one.
var unitNotes = map[string]string{
	"U_N_I_X_TIMESTAMP": "The source model spells this `U_N_I_X_TIMESTAMP`, an " +
		"artefact of splitting the VSS name \"UNIX Timestamp\" on case.",
	"MILES_PER_US_GALLON_DEPRECATED": "VSS retains this alongside " +
		"UNIT_MILES_PER_US_GALLON for compatibility; prefer that one.",
}

// quantity is one quantity kind and the units belonging to it.
type quantity struct {
	kind  string   // the source model's name, e.g. "angular-speed"
	enum  string   // the source enum, e.g. AngularSpeedUnitEnum
	units []string // its unit values, in declaration order
}

// quantities extracts the unit vocabulary from the parsed specification.
func quantities(defs []Def) []quantity {
	var out []quantity
	for _, d := range defs {
		if d.Kind != "enum" || !isUnitEnum(d.Name) {
			continue
		}
		q := quantity{enum: d.Name, kind: d.Doc}
		// The description reads: Units for "angular-speed"
		if i := strings.Index(d.Doc, "\""); i >= 0 {
			if j := strings.Index(d.Doc[i+1:], "\""); j >= 0 {
				q.kind = d.Doc[i+1 : i+1+j]
			}
		}
		for _, v := range d.Values {
			q.units = append(q.units, v.Name)
		}
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].enum < out[j].enum })
	return out
}

// vocabHeader renders the licence and file options for a vocabulary file.
func vocabHeader(spec Spec, class string, imports []string) string {
	sb := &strings.Builder{}
	sb.WriteString(spec.banner())
	sb.WriteString("syntax = \"proto3\";\n\npackage " + vocabPackage + ";\n\n")
	for _, i := range imports {
		sb.WriteString("import \"" + i + "\";\n")
	}
	if len(imports) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("option java_multiple_files = true;\n")
	sb.WriteString(fmt.Sprintf("option java_outer_classname = %q;\n", class))
	sb.WriteString("option java_package = \"" + javaBase + ".vss." + vocabName + ".v1\";\n\n")
	return sb.String()
}

// generateVocab writes the four vocabulary files.
func generateVocab(spec Spec, defs []Def, out string) (int, error) {
	dir := filepath.Join(out, "vss", vocabName, "v1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	qs := quantities(defs)
	files := map[string]string{
		"unit.proto":          renderUnitEnum(spec, qs),
		"quantity_kind.proto": renderQuantityKind(spec, qs),
		"options.proto":       vocabOptions(spec),
		"annotations.proto":   vocabAnnotations(spec),
	}
	for base, body := range files {
		if err := os.WriteFile(filepath.Join(dir, base), []byte(body), 0o644); err != nil {
			return 0, err
		}
	}
	// unitSymbols is read when rendering field comments, so it is filled from
	// the same table the enum is rendered from rather than duplicated.
	for _, q := range qs {
		for _, u := range q.units {
			unitSymbols["UNIT_"+normaliseUnit(u)] = unitSymbolTable[u]
		}
	}
	return len(files), nil
}

// renderUnitEnum writes unit.proto.
func renderUnitEnum(spec Spec, qs []quantity) string {
	sb := &strings.Builder{}
	sb.WriteString(vocabHeader(spec, "UnitProto", nil))
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
			sb.WriteString(fmt.Sprintf("  UNIT_%s = %d;\n", normaliseUnit(u), n))
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

// renderQuantityKind writes quantity_kind.proto.
func renderQuantityKind(spec Spec, qs []quantity) string {
	sb := &strings.Builder{}
	sb.WriteString(vocabHeader(spec, "QuantityKindProto", nil))
	sb.WriteString(docBlock("QuantityKind is the physical quantity a Unit measures.\n\n"+
		"These are the groups the source model declares as separate unit enums. "+
		"Unit flattens them into one; this names the group a given unit belongs "+
		"to, so a generator can tell which conversions are meaningful.\n\n"+
		"Reference: COVESA VSS quantity catalogue.\n"+
		"https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/quantities.yaml", ""))
	sb.WriteString("enum QuantityKind {\n  // Not specified.\n  QUANTITY_KIND_UNSPECIFIED = 0;\n\n")
	for i, q := range qs {
		var syms []string
		for _, u := range q.units {
			if s := unitSymbolTable[u]; s != "" {
				syms = append(syms, s)
			}
		}
		sb.WriteString(docBlock("\""+q.kind+"\": "+strings.Join(syms, ", ")+".", "  "))
		sb.WriteString(fmt.Sprintf("  QUANTITY_KIND_%s = %d;\n\n",
			strings.ToUpper(strings.ReplaceAll(q.kind, "-", "_")), i+1))
	}
	trimTrailingBlank(sb)
	sb.WriteString("}\n")
	return sb.String()
}
