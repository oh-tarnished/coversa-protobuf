// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

package main

// renames.go is the catalogue: every place VSS vocabulary and an AIP rule
// disagree, and the resolution chosen for it.
//
// Kept apart from the mapping logic in types.go because it is data, not code
// -- it is the table docs/conventions.md documents, and it is what a
// specification bump most often needs edited. Every entry here has a row
// there; add to both or neither.
import "strings"

// fieldRenames resolves the field names where VSS vocabulary and an AIP rule
// disagree. Keyed "OwnerType.sdlFieldName". Every entry is also a row in the
// catalogue at the foot of docs/conventions.md; add to both or neither.
var fieldRenames = map[string]string{
	// AIP-216 reserves `state` and `status` for server-owned lifecycle. Both
	// of these are physical positions of a mechanism, not a lifecycle.
	"Convertible.status": "roof_position",
	"Massage.status":     "activation",

	// AIP-142 reads a bare `time` as a Timestamp field. This one *is* a
	// timestamp, so it keeps the type and takes the name VSS's own comment
	// uses: "Time for next charging-related action".
	"Timer.time": "action_time",

	// AIP-140 bans prepositions, so nothing may begin `after_`. VSS means
	// fuel economy over the current tank, which is what this says without one.
	"FuelSystem.afterRefuelingFuelEconomy": "current_tank_fuel_economy",

	// AIP-140 prepositions again: of, in, to, since, including.
	"CombustionEngine.numberOfCylinders":         "cylinder_count",
	"CombustionEngine.numberOfValvesPerCylinder": "cylinder_valve_count",
	"FuelSystem.consumptionSinceLastRefuel":      "current_tank_consumption",
	"FuelSystem.consumptionSinceStart":           "trip_consumption",
	"Service.distanceToService":                  "service_distance",
	"Service.timeToService":                      "service_duration",
	"Charging.timeToComplete":                    "completion_duration",
	"ElectricMotor.timeInUse":                    "usage_duration",
	"TractionBattery.stateOfCharge":              "charge_state",
	"TractionBattery.stateOfHealth":              "health_state",
	"Vehicle.widthIncludingMirrors":              "width_mirrors_included",
	"Vehicle.traveledDistanceSinceStart":         "trip_traveled_distance",

	// AIP-142 requires a Timestamp field to end in `_time`.
	"CurrentLocation.timestamp": "observation_time",

	// AIP-142 reads a `_date` suffix as a claim that the field is a
	// Timestamp, and these are calendar dates -- a Date message, no time and
	// no zone. The suffix is dropped and the type carries the meaning; each
	// reads as the noun the source model means.
	"VehicleIdentification.dateVehicleFirstRegistered": "first_registration",
	"VehicleIdentification.productionDate":             "production",
	"VehicleIdentification.purchaseDate":               "purchase",
	"VehicleIdentification.vehicleModelDate":           "model_release",
	"TractionBattery.productionDate":                   "production",

	// AIP-143 mandates the name `region_code` for a field holding a country
	// or region code from a standard list, which this is: ISO 3166-1 alpha-2.
	"Address.country": "region_code",

	// AIP-148 reserves a bare `id` for the server-assigned uid, which this
	// resource already has. VSS means the unit's own hardware identifier.
	"ControlUnit.id": "control_unit_id",

	// AIP-140 wants the abbreviation `id` over `identifier`, and AIP-148 then
	// rejects a field called `id`. VSS means the occupant's OAuth identity
	// claims, so the field says that instead and evades both.
	"Occupant.identifier": "identity",

	// A resource carries a server-assigned `name` and `uid` of its own, so a
	// source field of either name has to move aside. The two identifiers are
	// distinct and both are kept -- protobuf-rfc's rule 9 -- because the
	// source model's id arrives inside imported data and must survive a round
	// trip unchanged or every re-import duplicates the record.
	"Person.name": "name_components",
	"Person.id":   "vdm_uid",
}

// roundTripIDs are source fields that carry an originating system's own
// identifier. They are OPTIONAL and IMMUTABLE, never server-assigned.
var roundTripIDs = map[string]bool{"Person.id": true}

// reservedResourceFields are the names a resource's own AIP fields occupy. A
// source field landing on one is a collision that must be resolved explicitly
// in fieldRenames rather than silently overwritten.
var reservedResourceFields = map[string]bool{
	"name": true, "uid": true, "etag": true,
	"create_time": true, "update_time": true,
	"delete_time": true, "expire_time": true,
}

// typeRenames resolves message names where VSS vocabulary meets an AIP rule.
var typeRenames = map[string]string{
	// AIP-140 wants `Id` over `Identifier`; AIP-148 then reserves it. The
	// message holds OAuth issuer and subject claims, which is an identity.
	"Identifier": "Identity",
}

// durationFields are signals VSS states as a bare number of seconds or hours
// and this schema states as a google.protobuf.Duration.
//
// Converting is not cosmetic. AIP-140 forbids the prepositions all three of
// their VSS names contain, so each had to be renamed regardless, and a name
// ending `_duration` that is not a Duration is the sort of disagreement
// between name and type that AIP-142 exists to prevent.
var durationFields = map[string]bool{
	"Service.timeToService":   true,
	"Charging.timeToComplete": true,
	"ElectricMotor.timeInUse": true,
}

// reservedWords are the identifiers AIP-140 rejects because they are keywords
// in a language the generated code targets. Every VSS occurrence so far is a
// `switch` -- a branch of switch signals on a door, hood, trunk or shade --
// and each becomes `switch_control`.
var reservedWords = map[string]bool{
	"switch": true, "case": true, "class": true, "default": true,
	"import": true, "package": true, "return": true, "type": true,
	"func": true, "interface": true, "map": true, "range": true,
	"const": true, "var": true, "for": true, "if": true, "else": true,
	"break": true, "continue": true, "goto": true, "struct": true,
	"public": true, "private": true, "static": true, "final": true,
	"new": true, "this": true, "null": true, "true": true, "false": true,
}

// capnpReserved are the Cap'n Proto keywords an enum value must not become.
//
// Cap'n Proto scopes enumerants inside their enum and strips the shared
// prefix, so LOW_VOLTAGE_SYSTEM_STATE_ON arrives as `on` -- a keyword. The
// generator escapes it to `on_`, which Cap'n Proto then rejects in turn,
// because it forbids underscores in declaration names. Neither form compiles,
// so the value is renamed here instead: the collision is resolved in the
// schema this repository owns rather than in a target it does not.
var capnpReserved = map[string]bool{
	"annotation": true, "as": true, "const": true, "else": true,
	"embed": true, "enum": true, "extends": true, "group": true,
	"if": true, "import": true, "in": true, "interface": true,
	"of": true, "on": true, "struct": true, "then": true,
	"union": true, "using": true, "void": true,
}

// enumValueName renders one enum value, prefixed per AIP-126 and kept clear of
// the keywords a target would choke on.
func enumValueName(prefix, value string) string {
	if capnpReserved[strings.ToLower(value)] {
		// `_VALUE`, not an escape character: the name has to survive being
		// stripped back to `onValue`, which no target reserves.
		value += "_VALUE"
	}
	return prefix + "_" + screaming(value)
}

// scalarEnums replace a source enum with a documented scalar.
//
// AIP-143 <https://aip.dev/143> asks for a *string* holding the code, not an
// enum of every value: the list is maintained by a standards body, changes
// without reference to this schema, and a country missing from a generated
// enum is a country the API cannot express.
//
// It also removes a Cap'n Proto problem, which is how the issue surfaced.
// That target scopes enumerants inside the enum and strips the shared prefix,
// so COUNTRY_CODE_AS becomes `as` and COUNTRY_CODE_IN becomes `in` -- both
// reserved -- and the escape the generator reaches for, `as_`, is itself
// rejected because Cap'n Proto forbids underscores in declaration names. The
// enum could not be represented in one of this schema's own target formats.
var scalarEnums = map[string]struct {
	proto string
	rule  string
	doc   string
}{
	"CountryCode": {
		proto: "string",
		rule:  `(buf.validate.field).string.pattern = "^[A-Z]{2}$"`,
		doc: "An ISO 3166-1 alpha-2 country code, e.g. \"SE\".\n\n" +
			"A string rather than an enum, per AIP-143 <https://aip.dev/143>: the " +
			"code list belongs to ISO and changes without reference to this schema.",
	},
}

// integerRange is the validation range for a VSS integer width that protobuf
// has no exact type for. AIP-141 forbids unsigned types, so every unsigned VSS
// width widens to a signed protobuf one and the lost bound is restored here.
type integerRange struct {
	proto string
	lo    string
	hi    string
}

var integerWidths = map[string]integerRange{
	"Int8":   {"int32", "-128", "127"},
	"UInt8":  {"int32", "0", "255"},
	"Int16":  {"int32", "-32768", "32767"},
	"UInt16": {"int32", "0", "65535"},
	"UInt32": {"int64", "0", "4294967295"},
}
