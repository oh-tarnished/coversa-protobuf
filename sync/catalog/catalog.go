// Copyright 2026 The Protobuf Project authors.
// SPDX-License-Identifier: Apache-2.0

// Package catalog is the resolution table: every place VSS vocabulary and an
// AIP rule disagree, and the choice made about it.
//
// It is data, not logic. Everything here is a lookup the mapping code
// consults, kept apart from that code because this is what a specification
// bump most often needs edited, and because every entry is also a row in the
// catalogue at the foot of docs/conventions.md. Add to both or neither.
//
// Two rules are applied by *rule* rather than by table, so a later VSS
// release cannot introduce one unnoticed: reserved words, in [ReservedWord],
// and AIP-140's abbreviation table, in package naming.
package catalog

// FieldRenames resolves field names where VSS vocabulary meets an AIP rule.
// Keyed "OwnerType.sdlFieldName".
var FieldRenames = map[string]string{
	// AIP-216 reserves `state` and `status` for server-owned lifecycle. Both
	// of these are physical positions of a mechanism, not a lifecycle.
	"Convertible.status": "roof_position",
	"Massage.status":     "activation",

	// AIP-142 reads a bare `time` as a Timestamp field. This one *is* a
	// timestamp, so it keeps the type and takes the name VSS's own comment
	// uses: "Time for next charging-related action".
	"Timer.time": "action_time",

	// AIP-140 bans prepositions: of, in, to, since, after, including.
	"FuelSystem.afterRefuelingFuelEconomy":       "current_tank_fuel_economy",
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
	// no zone. The suffix is dropped and the type carries the meaning.
	"VehicleIdentification.dateVehicleFirstRegistered": "first_registration",
	"VehicleIdentification.productionDate":             "production",
	"VehicleIdentification.purchaseDate":               "purchase",
	"VehicleIdentification.vehicleModelDate":           "model_release",
	"TractionBattery.productionDate":                   "production",

	// AIP-143 mandates `region_code` for a field holding a code from a
	// standard list, which this is: ISO 3166-1 alpha-2.
	"Address.country": "region_code",

	// AIP-148 reserves a bare `id` for the server-assigned uid, which every
	// resource here already has. VSS means the unit's own hardware id.
	"ControlUnit.id": "control_unit_id",

	// AIP-140 wants the abbreviation `id` over `identifier`, and AIP-148 then
	// rejects a field called `id`. VSS means the occupant's OAuth identity
	// claims, so the field says that instead and evades both.
	"Occupant.identifier": "identity",

	// A resource carries a server-assigned `name` and `uid` of its own, so a
	// source field of either name has to move aside. Both identifiers are
	// kept -- see RoundTripIDs.
	"Person.name": "name_components",
	"Person.id":   "vdm_uid",
}

// TypeRenames resolves message names where VSS vocabulary meets an AIP rule.
var TypeRenames = map[string]string{
	// AIP-140 wants `Id` over `Identifier`; AIP-148 then reserves it. The
	// message holds OAuth issuer and subject claims, which is an identity.
	"Identifier": "Identity",
}

// RoundTripIDs are source fields carrying an originating system's own
// identifier, which are OPTIONAL and IMMUTABLE rather than server-assigned.
//
// Two identifiers, not one: `uid` is ours and `vdm_uid` is theirs. The
// imported value must survive a round trip unchanged, or every re-import
// duplicates the record.
var RoundTripIDs = map[string]bool{"Person.id": true}

// ReservedResourceFields are the names a resource's own AIP fields occupy.
//
// A source field landing on one is a build error naming the field, never a
// silent overwrite: the generator fails loudly or not at all.
var ReservedResourceFields = map[string]bool{
	"name": true, "uid": true, "etag": true,
	"create_time": true, "update_time": true,
	"delete_time": true, "expire_time": true,
}

// DurationFields are signals VSS states as a bare number of seconds or hours
// and this schema states as a google.protobuf.Duration.
//
// Not cosmetic: AIP-140 forbids the prepositions all three VSS names contain,
// so each had to be renamed regardless, and a field named `_duration` that is
// not a Duration is the disagreement between name and type that AIP-142
// exists to prevent.
var DurationFields = map[string]bool{
	"Service.timeToService":   true,
	"Charging.timeToComplete": true,
	"ElectricMotor.timeInUse": true,
}
