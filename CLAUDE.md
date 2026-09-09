# coversa-protobuf — working conditions

Protobuf types for COVESA's Vehicle Signal Specification and Vehicle Data
Model, and the FlatBuffers and Cap'n Proto schemas derived from them. Every
rule below is a hard constraint, not a preference. `.github/workflows/`
enforces the ones that can be enforced mechanically, one workflow per concern
-- `lint`, `sync`, `schema`, `generate`, `breaking`, `links` -- so a failure names what
broke. The rest are on you.

The conventions are adapted from
[protobuf-rfc](https://github.com/the-protobuf-project/protobuf-rfc), whose
numbering this file keeps so the two can be read side by side. Where a rule
differs, it says so and why.

## 1. No lint rule may be disabled

Not with an `except` list in `buf.yaml`, not with a `disabled_rules` entry in
an api-linter config, not with an inline `(-- api-linter: ... =disabled --)`
comment. There is no api-linter config file in this repo and there must not
be one.

If a linter objects, the code is wrong. Fix the code -- which here means
fixing **the generator**, never the emitted file.

When two linters genuinely contradict each other, do not except a rule --
narrow one tool's scope so they stop overlapping. That is why `buf.yaml` uses
`BASIC` rather than `STANDARD`: buf STANDARD wants `VehiclesService` and
`GetVehicleResponse` wrappers, AIP-131 wants `Vehicles` returning `Vehicle`,
and no code satisfies both. buf checks structure, api-linter checks API
design, and every rule in each selected category is enforced.

## 2. Nothing under `protobuf/` is written by hand

Both the `.proto` files and the `README.md` beside them are generated -- the
schema by `sync`, the reference by `docs`, from the same model rather than by
parsing the emitted protos back in.

Every `.proto` in this repository is emitted by `sync` from the two
specifications pinned in `sync/spec.yaml`: the vehicle tree from VSS's own
`.vspec`, everything around it from VDM's GraphQL SDL. Every file carries a
`DO NOT EDIT` banner naming both revisions. Editing one is not a small
shortcut, it is a change that the next `just sync` silently reverts.

A fix belongs in the generator, and usually in a named table there:

| Change | Where |
|---|---|
| a field name a linter rejects | `FieldRenames` in `sync/catalog` |
| a field name a linter flags but that is right as it stands | `AcceptedTraps`, same file |
| a message name a linter rejects | `TypeRenames`, same file |
| a signal better typed as a Duration | `DurationFields`, same file |
| an enum better modelled as a scalar | `ScalarEnums` in `sync/catalog` |
| a plural English gets wrong | `irregularPlurals` in `sync/naming` |
| which functional area a branch belongs to | `domains` in `sync/model` |
| a value type shared across packages | `sharedValueTypes` in `sync/model` |
| which specification revisions to read | `sync/spec.yaml`, then `just sync` |

Every entry keyed by a fully qualified VSS name — the specification's own
identity for a signal, unambiguous where a bare field name is not. An entry
naming a signal the specification no longer has is a build error, not a
rename that quietly stops applying: see `model.CheckCatalogue`.

How the generator itself is laid out -- one package per stage, the 200-line
cap, and why there is a `go.work` -- is in `docs/generator.md`.

@docs/generator.md

This replaces protobuf-rfc's rule 2, the 250-line cap, which cannot apply to a
generated file: protobuf cannot continue a message across files, and a
generated file cannot be split by hand anyway. What that cap was really asking
for is enforced directly instead: **one message per file**, plus its enums,
with `messages.proto` the single exception because a request and its response
are meaningless apart. Both line caps, and what is exempt from them, are in
`docs/generator.md`.

## 3. AIP is the convention

api-linter must report zero violations across all 270 files. In practice that
means:

- `option java_package`, `java_outer_classname`, `java_multiple_files` on
  every file -- api-linter mandates all three (AIP-191)
- **no `go_package`**: the Go import path depends on where a consumer
  generates, so consumers set it through managed mode. Nothing forces it, so
  nothing is committed. The asymmetry with Java is not a preference --
  api-linter requires one and not the other, and rule 1 forbids excepting it
- a comment on every message, field, enum, enum value, service and RPC. VSS
  documents nearly everything, and the generator synthesises the rest from the
  declaration's own name; see `describe.Summary`
- `(google.api.field_behavior)` on every field
- `(google.api.resource)` on every resource, `(google.api.resource_reference)`
  on every field naming one
- `(google.api.http)` on every RPC, `(google.api.method_signature)` on the
  standard methods
- `int32`, never `uint32` (AIP-141). VSS's unsigned widths widen to a signed
  protobuf type and the lost bound is restored as a `buf.validate` range
- messages before top-level enums; services before messages
- a `uid` field carries `(google.api.field_info).format = UUID4` (AIP-148)

## 4. No cross-package message references

AIP-215 forbids a field from referencing a message in another proto package,
and exempts only `google.*`. A package must therefore contain everything it
references, which is what makes the branch split in rule 6 possible rather
than merely tidy.

Two consequences, and neither is a workaround:

- A **child resource** is not a field. `Vehicle` has no `cabin` field; the
  cabin is at `vehicles/{vehicle}/cabin` and is fetched from `Cabins`.
- A **genuine association** is a resource name. `ChargingSession.vehicle` is
  a `string` carrying `vehicles/{vehicle}`, with a
  `(google.api.resource_reference)` naming what it points at.

`google.*` is the one exemption, and this repository **does not take it** --
the single place it differs from protobuf-rfc's rule 4. `google.type.Date`
and `google.type.PostalAddress` were both adopted and both withdrawn, because
`protoc-gen-buffers` renders neither: a `PostalAddress` field vanished from
the emitted FlatBuffers table with no diagnostic. A value type used by more
than one package is instead **copied into each**, generated rather than
maintained. See `sharedValueTypes` and `docs/decisions.md`.

## 5. Custom annotations are allowed, and there is exactly one

protobuf-rfc bans them outright. This repository has one vocabulary,
`protobuf.covesa.vss.annotations.v1`, and the reason is in
`docs/decisions.md`: a VSS signal carries a unit, and protobuf has no place
to put it. `speed` is a `double`; nothing in the type says kilometres per
hour, and a consumer reading the number without the unit has read a different
quantity.

The GraphQL source states it as a field argument. Protobuf has no field
arguments, and a comment is not readable by the generators that consume this
schema, so it is an option -- and restated in the comment for a human. That
is the whole licence: **provenance still goes in comments**, and no second
vocabulary may be added without the same argument.

## 6. Layout

```
protobuf/covesa/<spec>/<domain>/<resource>/v1/<file>.proto
    package protobuf.covesa.<spec>.<domain>.<resource>.v1
```

`<spec>` is `vss` for the vehicle tree or `vdm` for what COVESA defines
outside it -- people, charging stations, sessions. `<domain>` is the
functional area from the `domains` table: motion, propulsion, interior,
exterior, assistance, location, platform. `vehicle` and the `vdm` packages
have no domain, since a root belongs to no area and the VDM set is small.

The annotation vocabulary is at `vss/annotations/v1`, **named** rather than
sitting at `vss/v1`. A bare version directory beside the domain directories
reads as a mistake, and `protobuf.covesa.vss.v1` would be a proper prefix of
`protobuf.covesa.vss.interior.cabin.v1` -- a resolution hazard, not a
cosmetic one.

The buf module is rooted at the **repository root**, so `protobuf/` is itself
the first package segment and `PACKAGE_DIRECTORY_MATCH` holds with no
redundant directory inside it. `build/` is excluded from the module; it holds
exported copies of googleapis and protovalidate for the schema step.

## 7. Every resource has full CRUD, and the shape follows the source model

A message with `google.api.resource` gets the standard methods in its
package's `service.proto`. Which methods depends on how many the parent has,
and the source model decides:

- A branch a vehicle holds **many** of -- seats, doors, control units -- is a
  collection: `Get`, `List`, `Create`, `Update`, `Delete`, plus AIP-164
  `Undelete`. 24 of them.
- A branch it holds **one** of -- the cabin, the powertrain -- is a
  singleton, AIP-156: `Get` and `Update` only. Nothing can create a second
  cabin or delete the only one. 23 of them.

A singleton having four fewer methods is not an exception to full CRUD.
AIP-156 defines the complete method set for a singleton and `Create` and
`Delete` are not in it.

**A branch with `instances` is a resource**, wherever it sits in the tree.
That declaration is VSS stating that its members have identity -- a seat is
Row1/DriverSide -- and a thing with identity is addressable. The axes are the
resource's id, not fields on it: `vehicles/{v}/seats/row1DriverSide`, with the
axes recorded in `codec/manifest.json` so an id can be taken back apart.

**A branch VSS names the same way twice is qualified by its path.** `Axle`
occurs three times and they are three different resources, so each takes as
much of its path as it needs: `chassis_axle`, `brake_axle`,
`suspension_axle`. See `model.disambiguate` and `docs/decisions.md`.

## 8. A signal's VSS kind decides whether it can be written

VSS marks each signal an attribute, a sensor or an actuator. Only an actuator
is writable: the other two are `OUTPUT_ONLY`, because a vehicle reports them
and no API call sets them. An `update_mask` naming one is rejected rather
than silently ignored.

## 9. The specification revision is pinned and dated

`sync/spec.yaml` names, for each of the two sources, the upstream commit this
schema was generated from and the date of that commit, `YYYY-MM-DD`, the way
MCP versions its specification. Both are stamped into every generated file's
banner, so "which specifications is this from?" is answerable from any single
`.proto` a consumer is holding. Both checkouts are submodules under
`modules/`, which holds every upstream this repository reads and nothing it
writes.

Bumping a pin is a deliberate act, not a side effect of pulling: move the
checkout, edit the pin, run `just sync`, and record what moved in
`docs/spec.md`. `just spec` checks each pin against its working tree --
commit, date, and a clean tree -- because a stale pin puts a revision into 270
banners that the files were never generated from, and nothing else can see
that is false.

@docs/spec.md

## Rules 10-15

Identifiers, the AIP naming traps and the full rename catalogue, unit handling
and the two annotation systems are in `docs/conventions.md`, imported below
and carrying the same weight; the split is length.

@docs/conventions.md

## Where to look things up

`docs/references.md` is a checked link index: the AIPs and their linter rule
pages, buf and protovalidate docs, the COVESA specifications this schema
models, and the two upstream projects it is built on. **Follow the link
rather than working from memory** -- AIP rule semantics have been guessed
wrong here more than once.

@docs/references.md

## Before you finish

Run what CI runs:

```sh
just ci
```

which runs, in order: the spec pin check, `buf format --diff --exit-code`,
`buf lint`, `buf build`, `api-linter`, the line cap, a regenerate-and-diff, the
two schema targets, their compilers, the ordinal ledger and the `schema/` diff. Each is also
a workflow under `.github/workflows/` -- read the steps there rather than
trusting this list to stay current.

`scripts/compile-schema.sh` is not redundant with `buf build`; why, and the
VS Code linter extension's disagreement with the CLI, are in
`docs/decisions.md` and `docs/references.md`.
