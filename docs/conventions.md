# Conventions

The second half of the working rules. `CLAUDE.md` holds rules 1-9 (structure,
layout and the spec pin) and imports this file; these are the ones about what goes *inside*
a file. Same standing as the others -- none is advisory.

## 10. Two identifiers, not one

`uid` is ours: OUTPUT_ONLY, server-assigned, `(google.api.field_info).format
= UUID4` because AIP-148 requires it.

Where the source model carries its own identity property, it gets a separate
field marked OPTIONAL and IMMUTABLE. `Person.id` becomes `vdm_uid`: it
arrives inside imported data, is whatever the originating system chose, and
must survive a round trip unchanged or every re-import duplicates the record.
Do not conflate the two.

The generator enforces this. A source field whose name lands on one of a
resource's own AIP fields -- `name`, `uid`, `etag`, the four timestamps --
is a build error naming the field and telling you to add a rename, rather
than a silent overwrite. See `reservedResourceFields`.

## 11. AIP naming traps

VSS's term for a signal is frequently unusable as a field name. Rename the
field, keep the source term in the comment, and add a row to the catalogue at
the foot of this file.

The recurring ones: AIP-122 bans the `_name` suffix (only `display_name`,
`given_name`, `family_name` survive) and a field called `name` makes
api-linter treat the message as a resource; AIP-140 bans prepositions, so
nothing may contain `of`, `in`, `to`, `since` or `including`; AIP-140 also
rejects a name that is a keyword in a common target language; AIP-141 forbids
unsigned integers; AIP-142 reads a `_time` or `_date` suffix as a claim that
the field is a Timestamp; AIP-143 mandates `region_code`; AIP-148 reserves a
bare `id`; and AIP-216 reserves `state` and `status` for server-owned
lifecycle.

Two of these are applied **by rule rather than by list**, so a later VSS
release cannot introduce one unnoticed: reserved words (`switch` becomes
`switch_control`) and AIP-140's abbreviation table (`configuration` becomes
`config`). The rest are the table below.

Resource **patterns** trap differently. AIP-123 requires a name to alternate
collection and identifier, and a singleton's name ends in a literal -- so a
child of a singleton cannot hang beneath it without putting two literals in a
row. A seat's name is therefore `vehicles/{vehicle}/seats/{seat}`, not
`.../cabin/seats/{seat}`. Nothing is lost: a cabin has exactly one
occurrence, so a `cabin` segment identifies nothing, and the full VSS path is
in the branch annotation's `fqn` either way. See `namingParent`.

## 12. Every citation carries its link

These comments become the generated documentation in every target language,
so a citation without a URL is a lookup the reader does by hand. AIP mentions
get `AIP-131 <https://aip.dev/131>`; VSS references name the fully qualified
signal and link the specification.

## 13. Units are an annotation, and a fixed one

A VSS signal states its unit as a GraphQL field argument with a default:
`speed(unit: VelocityUnitEnum = KILOMETER_PER_HOUR)`. Protobuf has neither
field arguments nor a place to put a unit, so:

- the schema **pins** the source model's default and records it in
  `(protobuf.covesa.vss.annotations.v1.signal)`, alongside the quantity kind,
  the VSS element type and the fully qualified name;
- the same facts are restated in the field's doc comment, because the option
  is invisible in generated documentation.

Pinned, not negotiated. Putting the unit on the wire beside the value would
let two producers of the same field disagree about what the number means,
which is the one failure the source model is careful to prevent.

A signal whose unit is `DatetimeUnitEnum` is not annotated with a unit at
all: it becomes a `google.protobuf.Timestamp` or a `Date`, and the type says
it better than the annotation would.

## 14. Two annotations, two jobs

**`google.api.field_behavior`** declares intent, and api-linter requires it on
every field. `IDENTIFIER` on a resource `name`, `OUTPUT_ONLY` for anything the
server sets and for every VSS attribute and sensor (rule 8), `IMMUTABLE` where
a value may be set once, `OPTIONAL` for an actuator.

**`buf.validate`** (protovalidate) enforces. Constraints come from the source
model, not from taste: a `@range(min: 0, max: 100)` becomes a numeric rule,
and an unsigned VSS width becomes the bound that AIP-141 made the type lose.
Where both apply they are **intersected** into one rule, because protobuf
accepts a single rule per field and emitting two is a compile error.

One trap: a constraint on a possibly-unset field must carry
`(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE`, or an empty string
fails `string.uuid` and every create request is rejected.

## 15. The generator fails loudly or not at all

`sync` parses only the SDL subset `vdm/spec` actually uses and errors
on anything else. That is deliberate: the input is a generated, regular
corpus, and a parser that quietly accepts an unfamiliar shape produces a
schema that is wrong in a way no linter catches.

The same applies downstream. An unknown type is an error, not a skipped
field; a name collision with a resource's own fields is an error, not an
overwrite. Run `just survey` after a specification bump:
it reports what was parsed and every field name that trips a known AIP rule,
which is how the catalogue below was built and how it should be rechecked.

## The catalogue

VSS vocabulary and AIP rules disagree in the places below. Applied by rule,
not listed: `switch` -> `switch_control` (AIP-140 reserved words) in eight
branches, and `configuration` -> `config` (AIP-140 abbreviations).

| Source property | Field | Why |
|---|---|---|
| `Person.name` | `name_components` | a field called `name` makes api-linter treat the message as a resource |
| `Person.id` | `vdm_uid` | AIP-148 reserves a bare `id`; rule 9's round-trip identifier |
| `Occupant.identifier` | `identity` | AIP-140 wants `id`, which AIP-148 then rejects; the field holds OAuth claims |
| `ControlUnit.id` | `control_unit_id` | AIP-148 again; VSS means the unit's own hardware id |
| `Convertible.status` | `roof_position` | AIP-216 reserves `status`; this is a physical position |
| `Massage.status` | `activation` | same |
| `*.Status` enums | `*State` | AIP-216 prefers `State` over `Status` |
| `Timer.time` | `action_time` | AIP-142 reads a bare `time` as a Timestamp field |
| `CurrentLocation.timestamp` | `observation_time` | AIP-142 requires a Timestamp field to end in `_time` |
| `FuelSystem.afterRefuelingFuelEconomy` | `current_tank_fuel_economy` | AIP-140 bans prepositions |
| `FuelSystem.consumptionSinceLastRefuel` | `current_tank_consumption` | same |
| `FuelSystem.consumptionSinceStart` | `trip_consumption` | same |
| `CombustionEngine.numberOfCylinders` | `cylinder_count` | same |
| `CombustionEngine.numberOfValvesPerCylinder` | `cylinder_valve_count` | same |
| `Service.distanceToService` | `service_distance` | same |
| `Service.timeToService` | `service_duration` | same, and becomes a Duration |
| `Charging.timeToComplete` | `completion_duration` | same |
| `ElectricMotor.timeInUse` | `usage_duration` | same |
| `TractionBattery.stateOfCharge` | `charge_state` | same |
| `TractionBattery.stateOfHealth` | `health_state` | same |
| `Vehicle.widthIncludingMirrors` | `width_mirrors_included` | same |
| `Vehicle.traveledDistanceSinceStart` | `trip_traveled_distance` | same |
| `*.productionDate`, `purchaseDate`, `vehicleModelDate` | `production`, `purchase`, `model_release` | AIP-142 reads `_date` as a claim the field is a Timestamp; these are `Date` |
| `VehicleIdentification.dateVehicleFirstRegistered` | `first_registration` | same |
| `Address.country` | `region_code` | AIP-143 mandates the name for an ISO 3166-1 value |
| `CountryCode` enum | `string` | AIP-143 wants the code as a string; also unrepresentable in Cap'n Proto, see `scalarEnums` |
| `LOW_VOLTAGE_SYSTEM_STATE_ON` | `..._ON_VALUE` | Cap'n Proto strips the enum prefix, and `on` is one of its keywords |
