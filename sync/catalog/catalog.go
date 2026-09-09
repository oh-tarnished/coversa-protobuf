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

// FieldRenames resolves field names where source vocabulary meets an AIP
// rule. Keyed by fully qualified name — the specification's own identity for
// a signal, which is unambiguous where a bare field name is not.
var FieldRenames = map[string]string{
	// AIP-216 reserves `state` and `status` for server-owned lifecycle. Both
	// of these are physical positions of a mechanism, not a lifecycle.
	"Vehicle.Cabin.Convertible.Status":  "roof_position",
	"Vehicle.Cabin.Seat.Massage.Status": "activation",

	// AIP-142 reads a bare `time` as a Timestamp field. This one *is* a
	// timestamp, so it keeps the type and takes the name the specification's
	// own comment uses: "Time for next charging-related action".
	"Vehicle.Powertrain.TractionBattery.Charging.Timer.Time": "action_time",

	// AIP-142 requires a Timestamp field to end in `_time`.
	"Vehicle.CurrentLocation.Timestamp": "observation_time",

	// AIP-140 bans prepositions: of, in, to, since, after, including.
	"Vehicle.Powertrain.FuelSystem.AfterRefuelingFuelEconomy":       "current_tank_fuel_economy",
	"Vehicle.Powertrain.FuelSystem.ConsumptionSinceLastRefuel":      "current_tank_consumption",
	"Vehicle.Powertrain.FuelSystem.ConsumptionSinceStart":           "trip_consumption",
	"Vehicle.Powertrain.CombustionEngine.NumberOfCylinders":         "cylinder_count",
	"Vehicle.Powertrain.CombustionEngine.NumberOfValvesPerCylinder": "cylinder_valve_count",
	"Vehicle.Powertrain.TractionBattery.StateOfCharge":              "charge_state",
	"Vehicle.Powertrain.TractionBattery.StateOfHealth":              "health_state",
	"Vehicle.Powertrain.TractionBattery.Charging.TimeToComplete":    "completion_duration",
	"Vehicle.Powertrain.ElectricMotor.TimeInUse":                    "usage_duration",
	"Vehicle.Service.DistanceToService":                             "service_distance",
	"Vehicle.Service.TimeToService":                                 "service_duration",
	"Vehicle.WidthIncludingMirrors":                                 "width_mirrors_included",
	"Vehicle.TraveledDistanceSinceStart":                            "trip_traveled_distance",

	// AIP-142 reads a `_date` suffix as a claim that the field is a
	// Timestamp, and these are calendar dates -- a Date message, no time and
	// no zone. The suffix is dropped and the type carries the meaning.
	"Vehicle.VehicleIdentification.DateVehicleFirstRegistered": "first_registration",
	"Vehicle.VehicleIdentification.ProductionDate":             "production",
	"Vehicle.VehicleIdentification.PurchaseDate":               "purchase",
	"Vehicle.VehicleIdentification.VehicleModelDate":           "model_release",
	"Vehicle.Powertrain.TractionBattery.ProductionDate":        "production",

	// AIP-216 reserves `state` for a server-owned lifecycle. A postal address
	// means the largest subdivision below the country, which is what
	// google.type.PostalAddress calls administrative_area -- a state in the
	// US, a prefecture in Japan, nothing at all in Singapore.
	"Person.HomeAddress.State":                      "administrative_area",
	"ChargingStation.Location.State":                "administrative_area",
	"ChargingStation.ChargingPoints.Location.State": "administrative_area",

	// AIP-143 mandates `region_code` for a field holding a code from a
	// standard list, which this is: ISO 3166-1 alpha-2.
	"Person.HomeAddress.Country":                      "region_code",
	"ChargingStation.Location.Country":                "region_code",
	"ChargingStation.ChargingPoints.Location.Country": "region_code",

	// AIP-148 reserves a bare `id` for the server-assigned uid, which every
	// resource here already has. VSS means the unit's own hardware id.
	"Vehicle.ControlUnit.ID": "control_unit_id",

	// AIP-140 wants the abbreviation `id` over `identifier`, and AIP-148 then
	// rejects a field called `id`. VSS means the occupant's OAuth identity
	// claims, so the field says that instead and evades both.
	"Vehicle.Occupant.Identifier": "identity",

	// A resource carries a server-assigned `name` and `uid` of its own, so a
	// source field of either name has to move aside. Both identifiers are
	// kept -- the imported one must survive a round trip unchanged, or every
	// re-import duplicates the record.
	"Person.Name": "name_components",
	"Person.Id":   "vdm_uid",
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
var RoundTripIDs = map[string]bool{"Person.Id": true}

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
	"Vehicle.Service.TimeToService":                              true,
	"Vehicle.Powertrain.TractionBattery.Charging.TimeToComplete": true,
	"Vehicle.Powertrain.ElectricMotor.TimeInUse":                 true,
}

// AcceptedTraps are field names that hit a known AIP rule and keep the name
// anyway, each with the reason.
//
// A rename is not always the right answer. AIP-122 objects to an `_id` suffix
// because a field naming another resource should carry that resource's *name*;
// where the value is an identifier issued by something outside this API, there
// is no resource name to carry and the suffix is accurate.
//
// Recorded rather than left to a reader's judgement: `just survey` lists every
// unresolved trap, and a trap nobody decided about is indistinguishable from
// one nobody noticed.
var AcceptedTraps = map[string]string{
	"Vehicle.Powertrain.TractionBattery.Charging.EvseId": "the identifier of " +
		"the charging equipment, issued by the operator. Not a resource this " +
		"API defines, so there is no resource name to carry instead.",
}
