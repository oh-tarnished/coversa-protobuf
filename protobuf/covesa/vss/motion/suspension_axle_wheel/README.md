# Wheel

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.MotionManagement.Suspension.Axle.Wheel-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-4-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.motion.suspension__axle__wheel.v1-444)](v1/)

MotionManagement signals for a specific wheel.

```
vehicles/{vehicle}/suspensionAxles/{suspension_axle}/wheels/{wheel}
```

## Where it sits

```mermaid
flowchart LR
  P["SuspensionAxle<br/><code>vehicles/{vehicle}/suspensionAxles/{suspension_axle}</code>"]
  R["Wheel<br/><code>…/wheels/{wheel}</code>"]

  P -->|"many, each identified"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

## Example

A wheel as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}/wheels/wheel-prza36g0k81eez2w",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "dampingForceTarget": 42,
  "dampingRateTarget": 33,
  "dampingForce": 42,
  "dampingRate": 33
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingForceTarget": 42,
  "Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingRateTarget": 33,
  "Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingForce": 42,
  "Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingRate": 33
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

Writing to a wheel, and the one thing that will fail:

```mermaid
sequenceDiagram
  participant C as Client
  participant A as Wheels
  participant V as Vehicle

  Note over C,V: damping_force_target is an actuator — the API is the source
  C->>A: PATCH …?updateMask=dampingForceTarget
  A->>V: set damping_force_target
  V-->>A: acknowledged
  A-->>C: 200 OK

  Note over C,V: damping_force is a sensor — the vehicle is the source
  C->>A: PATCH …?updateMask=dampingForce
  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field
```

```http
PATCH /v1/vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}/wheels/wheel-prza36g0k81eez2w?updateMask=dampingForceTarget
{ "dampingForceTarget": 42 }
```

The rejection is the schema working, not an inconvenience: VSS marks
`damping_force` a sensor, so the vehicle is the only thing that can set it. A
mask naming it fails rather than being silently dropped, so a caller that
believed it had written the value finds out immediately.

## Resource names

Every wheel is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-41: "suspensionAxles"
42: "/"
43-59: "{suspension_axle}"
60: "/"
61-66: "wheels"
67: "/"
68-89: "wheel-prza36g0k81eez2w"
```

90 characters, alternating, which is what [AIP-123](https://aip.dev/123)
requires. An identifier is 80 random bits in Crockford base32, prefixed with
the resource's singular so the name opens with a letter — the randomness
half of a ULID with the timestamp half removed, because a ULID's leading
characters *are* a timestamp and a resource name should not publish when the
record was created. The vehicle is the exception: its identifier is its VIN,
which is already unique and already meaningful, the case AIP-122 prefers.

Rather than splitting that string by hand, use
[`resourcename`](https://github.com/the-protobuf-project/resourcename), which
implements AIP-122 templates in Go, Python, Rust, TypeScript, Swift and C:

```go
t := resourcename.ResourceTemplate("vehicles/{vehicle}/suspensionAxles/{suspension_axle}/wheels/{wheel}")
t.Parse("vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}/wheels/wheel-prza36g0k81eez2w")
// map[vehicle:wvwzzz1jz3w000001 suspensionAxle:suspensionAxle-bnfckq28mrrrjw6x wheel:wheel-prza36g0k81eez2w]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/suspensionAxles/{suspension_axle}/wheels/{wheel}")
t.parse("vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}/wheels/wheel-prza36g0k81eez2w")
# {'vehicle': 'wvwzzz1jz3w000001', 'suspensionAxle': 'suspensionAxle-bnfckq28mrrrjw6x', 'wheel': 'wheel-prza36g0k81eez2w'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

11 fields: 7 AIP identity and lifecycle, 4 VSS signals. Field numbers 8–15 are reserved for identity fields a later revision may add, so adding one never renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `damping_force_target` | `int32` | `N` | `Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingForceTarget` | Damping force request at given wheel. Signal orientation according ISO8855, meaning positive force is pointing upwards. |
| `damping_rate_target` | `int32` | `percent` | `Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingRateTarget` | Damping rate request at given wheel. 0% = lowest possible damping rate 100% = highest possible damping rate |

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `damping_force` | `int32` | `N` | `Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingForce` | Actuated damping force at given wheel. Signal orientation according ISO8855, meaning positive force is pointing upwards. |
| `damping_rate` | `int32` | `percent` | `Vehicle.MotionManagement.Suspension.Axle.Wheel.DampingRate` | Actuated damping rate at given wheel. 0% = lowest possible damping rate 100% = highest possible damping rate |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/suspensionAxles/{suspension_axle}/wheels/{wheel}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which wheel

```mermaid
flowchart LR
  A0["Left"]
  A1["Right"]
```

VSS expands this branch across Left, Right, so the combination names one wheel
— which is what makes it a resource rather than a repeated field. The axes
are recorded in [`codec/manifest.json`](../../../../../codec/manifest.json),
which is the only thing that says how to map an id back to a VSS path.

## Methods

A collection, so the full set: **`Get`** · **`List`** · **`Create`** · **`Update`** · **`Delete`** · **`Undelete`**.

```mermaid
stateDiagram-v2
  [*] --> Live: Create
  Live --> Live: Update · only actuators
  Live --> Deleted: Delete · soft
  Deleted --> Live: Undelete
  Deleted --> [*]: expire_time passes
```

```http
GET    /v1/{name=vehicles/*/suspensionAxles/*/wheels/*}
PATCH  /v1/{wheel.name=vehicles/*/suspensionAxles/*/wheels/*}
GET    /v1/{parent=vehicles/*/suspensionAxles/*}/wheels
POST   /v1/{parent=vehicles/*/suspensionAxles/*}/wheels
DELETE /v1/{name=vehicles/*/suspensionAxles/*/wheels/*}
POST   /v1/{name=vehicles/*/suspensionAxles/*/wheels/*}:undelete
```

A deleted wheel is still returned by `Get` and hidden from `List` unless
`show_deleted` is set — which is what makes `Undelete` meaningful.

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`wheel.proto`](v1/wheel.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
