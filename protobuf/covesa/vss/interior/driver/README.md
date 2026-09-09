# Driver

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Driver-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-singleton-2B3172)](https://aip.dev/156)
[![RPCs](https://img.shields.io/badge/RPCs-2-555)](#methods)
[![signals](https://img.shields.io/badge/signals-6-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.interior.driver.v1-444)](v1/)

Driver data.

```
vehicles/{vehicle}/driver
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Driver<br/><code>…/{vehicle}/driver</code>"]

  P -->|"exactly one"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

## Example

A driver as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/driver",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "attentiveProbability": 33.3,
  "distractionLevel": 33.3,
  "fatigueLevel": 33.3,
  "heartRate": 42,
  "isEyesOnRoad": true
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Driver.AttentiveProbability": 33.3,
  "Vehicle.Driver.DistractionLevel": 33.3,
  "Vehicle.Driver.FatigueLevel": 33.3,
  "Vehicle.Driver.HeartRate": 42,
  "Vehicle.Driver.IsEyesOnRoad": true
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every driver is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-32: "driver"
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/driver")
t.Parse("vehicles/wvwzzz1jz3w000001/driver")
// map[vehicle:wvwzzz1jz3w000001]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/driver")
t.parse("vehicles/wvwzzz1jz3w000001/driver")
# {'vehicle': 'wvwzzz1jz3w000001'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

13 fields: 7 AIP identity and lifecycle, 6 VSS signals. Field numbers 8–15 are reserved for identity fields a later revision may add, so adding one never renumbers a signal.

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `attentive_probability` | `double` | `percent` | `Vehicle.Driver.AttentiveProbability` | Probability of attentiveness of the driver. |
| `distraction_level` | `double` | `percent` | `Vehicle.Driver.DistractionLevel` | Distraction level of the driver, which can be evaluated by multiple factors e.g. driving situation, acoustical or optical signals inside the cockpit, ongoing phone calls. |
| `fatigue_level` | `double` | `percent` | `Vehicle.Driver.FatigueLevel` | Fatigue level of the driver, which can be evaluated by multiple factors e.g. trip time, behaviour of steering, eye status. |
| `heart_rate` | `int32` | `bpm` | `Vehicle.Driver.HeartRate` | Heart rate of the driver. |
| `is_eyes_on_road` | `bool` | — | `Vehicle.Driver.IsEyesOnRoad` | Has driver the eyes on road or not? |
| `is_hands_on_wheel` | `bool` | — | `Vehicle.Driver.IsHandsOnWheel` | Are the driver's hands on the steering wheel or not? |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/driver`, server-assigned |
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
GET    /v1/{name=vehicles/*/driver}
PATCH  /v1/{driver.name=vehicles/*/driver}
```

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`driver.proto`](v1/driver.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
