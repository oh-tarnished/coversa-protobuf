# Safety

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Safety-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-singleton-2B3172)](https://aip.dev/156)
[![RPCs](https://img.shields.io/badge/RPCs-2-555)](#methods)
[![signals](https://img.shields.io/badge/signals-5-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.assistance.safety.v1-444)](v1/)

Safety-related derived signals for crash detection and emergency response.

```
vehicles/{vehicle}/safety
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Safety<br/><code>…/{vehicle}/safety</code>"]

  P -->|"exactly one"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

## Example

A safety as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/safety",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isFire": true,
  "isSubmersed": true,
  "roadIcingState": "ROAD_ICING_STATE_NONE",
  "rollover": "ROLLOVER_NONE",
  "visibilityImpairment": "VISIBILITY_IMPAIRMENT_NONE"
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Safety.IsFire": true,
  "Vehicle.Safety.IsSubmersed": true,
  "Vehicle.Safety.RoadIcingState": "NONE",
  "Vehicle.Safety.Rollover": "NONE",
  "Vehicle.Safety.VisibilityImpairment": "NONE"
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every safety is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-32: "safety"
```

33 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/safety")
t.Parse("vehicles/wvwzzz1jz3w000001/safety")
// map[vehicle:wvwzzz1jz3w000001]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/safety")
t.parse("vehicles/wvwzzz1jz3w000001/safety")
# {'vehicle': 'wvwzzz1jz3w000001'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

12 fields: 7 AIP identity and lifecycle, 5 VSS signals. Field numbers 8–15
are reserved for identity fields a later revision may add, so adding one never
renumbers a signal.

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_fire` | `bool` | — | `Vehicle.Safety.IsFire` | Indicates whether a fire has been detected in or on the vehicle. |
| `is_submersed` | `bool` | — | `Vehicle.Safety.IsSubmersed` | Indicates whether the vehicle is partially or fully submersed in water. |
| `road_icing_state` | [`RoadIcingState`](#roadicingstate) | — | `Vehicle.Safety.RoadIcingState` | Indicates whether the vehicle is currently at risk of icing-related loss of control or has detected such conditions affecting traction. |
| `rollover` | [`Rollover`](#rollover) | — | `Vehicle.Safety.Rollover` | Indicates whether the vehicle has rolled over or is at risk of rollover. |
| `visibility_impairment` | [`VisibilityImpairment`](#visibilityimpairment) | — | `Vehicle.Safety.VisibilityImpairment` | Indicates whether reduced exterior visibility is currently creating a driving-safety risk, or such a condition has been detected. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/safety`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Methods

A singleton, so **`Get`** · **`Update`** only, per
[AIP-156](https://aip.dev/156). Nothing can create a second or delete the only
one — that is the complete method set for a singleton, not a reduced one.

```mermaid
stateDiagram-v2
  [*] --> Live: exists with its parent
  Live --> Live: Update · only actuators
```

```http
GET    /v1/{name=vehicles/*/safety}
PATCH  /v1/{safety.name=vehicles/*/safety}
```

## Enums

#### `Rollover`

Indicates whether the vehicle has rolled over or is at risk of rollover.

`UNSPECIFIED` · `UNKNOWN` · `NONE` · `RISK` · `DETECTED`
<sub>emitted as `ROLLOVER_UNKNOWN`, and so on</sub>

#### `VisibilityImpairment`

Indicates whether reduced exterior visibility is currently creating a
driving-safety risk, or such a condition has been detected.

`UNSPECIFIED` · `UNKNOWN` · `NONE` · `RISK` · `DETECTED`
<sub>emitted as `VISIBILITY_IMPAIRMENT_UNKNOWN`, and so on</sub>

#### `RoadIcingState`

Indicates whether the vehicle is currently at risk of icing-related loss of
control or has detected such conditions affecting traction.

`UNSPECIFIED` · `UNKNOWN` · `NONE` · `RISK` · `DETECTED`
<sub>emitted as `ROAD_ICING_STATE_UNKNOWN`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`safety.proto`](v1/safety.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
