// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package catalog

import "strings"

// reservedWords are identifiers AIP-140 rejects because they are keywords in
// a language the generated code targets.
//
// Applied by rule rather than by listing the eight `switch` fields VSS has
// today, so a new one in a later release is caught rather than shipped.
var reservedWords = map[string]bool{
	"switch": true, "case": true, "class": true, "default": true,
	"import": true, "package": true, "return": true, "type": true,
	"func": true, "interface": true, "map": true, "range": true,
	"const": true, "var": true, "for": true, "if": true, "else": true,
	"break": true, "continue": true, "goto": true, "struct": true,
	"public": true, "private": true, "static": true, "final": true,
	"new": true, "this": true, "null": true, "true": true, "false": true,
}

// ReservedWord reports whether a field name would collide with a keyword in a
// common target language. The caller appends `_control`, which is what every
// VSS occurrence -- a branch of switch signals on a door, hood, trunk or
// shade -- actually describes.
func ReservedWord(name string) bool { return reservedWords[name] }

// capnpReserved are the Cap'n Proto keywords an enum value must not become.
//
// Cap'n Proto scopes enumerants inside their enum and strips the shared
// prefix, so LOW_VOLTAGE_SYSTEM_STATE_ON arrives as `on` -- a keyword. The
// generator there escapes it to `on_`, which Cap'n Proto rejects in turn,
// because it forbids underscores in declaration names. Neither form compiles,
// so the collision is resolved in the schema this repository owns rather than
// in a target it does not.
var capnpReserved = map[string]bool{
	"annotation": true, "as": true, "const": true, "else": true,
	"embed": true, "enum": true, "extends": true, "group": true,
	"if": true, "import": true, "in": true, "interface": true,
	"of": true, "on": true, "struct": true, "then": true,
	"union": true, "using": true, "void": true,
}

// EnumValueSuffix returns the suffix an enum value needs to survive being
// stripped back to its bare name by a target that scopes enumerants.
//
// `_VALUE` rather than an escape character: the result has to be a legal
// declaration name in Cap'n Proto, which `on_` is not.
func EnumValueSuffix(value string) string {
	if capnpReserved[strings.ToLower(value)] {
		return "_VALUE"
	}
	return ""
}

// ScalarEnum is a source enum replaced by a documented scalar.
type ScalarEnum struct {
	// Proto is the protobuf type that replaces the enum.
	Proto string
	// Rule is the buf.validate constraint that keeps the scalar honest.
	Rule string
	// Doc is appended to the field's comment, saying what the scalar holds.
	Doc string
}

// ScalarEnums replace a source enum with a documented scalar.
//
// AIP-143 asks for a string holding the code, not an enum of every value: the
// list is maintained by a standards body, changes without reference to this
// schema, and a country missing from a generated enum is a country the API
// cannot express.
//
// It also removes a Cap'n Proto problem, which is how the issue surfaced.
// COUNTRY_CODE_AS becomes `as` and COUNTRY_CODE_IN becomes `in` -- both
// reserved -- so the enum could not be represented in one of this schema's
// own target formats at all.
var ScalarEnums = map[string]ScalarEnum{
	"CountryCode": {
		Proto: "string",
		Rule:  `(buf.validate.field).string.pattern = "^[A-Z]{2}$"`,
		Doc: "An ISO 3166-1 alpha-2 country code, e.g. \"SE\".\n\n" +
			"A string rather than an enum, per AIP-143 <https://aip.dev/143>: the " +
			"code list belongs to ISO and changes without reference to this schema.",
	},
}

// IsScalarEnum reports whether an enum is replaced by a scalar and therefore
// never emitted as a type.
func IsScalarEnum(name string) bool {
	_, ok := ScalarEnums[name]
	return ok
}

// IntegerWidth is the protobuf type and bounds for a VSS integer width.
//
// AIP-141 forbids unsigned protobuf types, so every unsigned VSS width widens
// to a signed one and the bound the type just lost is restored as a
// buf.validate range.
type IntegerWidth struct {
	Proto string
	Lo    string
	Hi    string
}

// IntegerWidths maps each VSS integer scalar onto its protobuf type.
var IntegerWidths = map[string]IntegerWidth{
	"Int8":   {"int32", "-128", "127"},
	"UInt8":  {"int32", "0", "255"},
	"Int16":  {"int32", "-32768", "32767"},
	"UInt16": {"int32", "0", "65535"},
	"UInt32": {"int64", "0", "4294967295"},
}
