# SuspensionAxle

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.MotionManagement.Suspension.Axle-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-1-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.motion.suspension__axle.v1-444)](v1/)

MotionManagement for suspension for a specific axle.

```
vehicles/{vehicle}/suspensionAxles/{suspension_axle}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["SuspensionAxle<br/><code>…/suspensionAxles/{suspension_axle}</code>"]
  C0["Wheel<br/><code>…/wheels/{wheel}</code>"]

  P -->|"many, each identified"| R
  R -->|"many, each identified"| C0

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P,C0 res;
```

In the specification this hangs beneath **MotionManagement**, but its name
hangs off **Vehicle**. AIP-123 requires a name to alternate collection and
identifier, and `motionManagement` is a singleton whose name ends in a literal
— two literals in a row is not a legal name. Nothing is lost:
motionManagement has one occurrence, so the segment would identify nothing,
and the full VSS path survives in the branch annotation.

## Example

A suspensionAxle as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "rollTorque": 42
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.MotionManagement.Suspension.Axle.RollTorque": 42
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every suspensionAxle is addressed by a name that alternates collection and
identifier, per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-41: "suspensionAxles"
42: "/"
43-59: "{suspension_axle}"
```

60 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/suspensionAxles/{suspension_axle}")
t.Parse("vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}")
// map[vehicle:wvwzzz1jz3w000001 suspensionAxle:suspensionAxle-bnfckq28mrrrjw6x]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/suspensionAxles/{suspension_axle}")
t.parse("vehicles/wvwzzz1jz3w000001/suspensionAxles/{suspension_axle}")
# {'vehicle': 'wvwzzz1jz3w000001', 'suspensionAxle': 'suspensionAxle-bnfckq28mrrrjw6x'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

8 fields: 7 AIP identity and lifecycle, 1 VSS signals. Field numbers 8–15
are reserved for identity fields a later revision may add, so adding one never
renumbers a signal.

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `roll_torque` | `int32` | `Nm` | `Vehicle.MotionManagement.Suspension.Axle.RollTorque` | Actuated roll torque on this axle by anti-roll actuator. Signal orientation according to ISO8855. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/suspensionAxles/{suspension_axle}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which suspensionAxle

```mermaid
flowchart LR
  A0["Row1"]
  A1["Row2"]
```

VSS expands this branch across Row1, Row2, so the combination names one
suspensionAxle — which is what makes it a resource rather than a repeated
field. The axes are recorded in
[`codec/manifest.json`](../../../../../codec/manifest.json), which is the only
thing that says how to map an id back to a VSS path.

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
GET    /v1/{name=vehicles/*/suspensionAxles/*}
PATCH  /v1/{suspensionAxle.name=vehicles/*/suspensionAxles/*}
GET    /v1/{parent=vehicles/*}/suspensionAxles
POST   /v1/{parent=vehicles/*}/suspensionAxles
DELETE /v1/{name=vehicles/*/suspensionAxles/*}
POST   /v1/{name=vehicles/*/suspensionAxles/*}:undelete
```

A deleted suspensionAxle is still returned by `Get` and hidden from `List`
unless `show_deleted` is set — which is what makes `Undelete` meaningful.

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`suspension_axle.proto`](v1/suspension_axle.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
