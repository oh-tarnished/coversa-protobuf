# Adas

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.ADAS-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-singleton-2B3172)](https://aip.dev/156)
[![RPCs](https://img.shields.io/badge/RPCs-2-555)](#methods)
[![signals](https://img.shields.io/badge/signals-12-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.assistance.adas.v1-444)](v1/)

All Advanced Driver Assist Systems data.

```
vehicles/{vehicle}/adas
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Adas<br/><code>…/{vehicle}/adas</code>"]
  E0["CruiseControl"]
  E1["LaneDepartureDetection"]
  E2["Abs"]
  E3["Tcs"]
  E4["Esc"]
  E5["RoadFriction"]
  E6["Ebd"]
  E7["Eba"]
  E8["Dms"]

  P -->|"exactly one"| R
  R --- E0
  R --- E1
  R --- E2
  R --- E3
  R --- E4
  R --- E5
  R --- E6
  R --- E7
  R --- E8

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
  class E0,E1,E2,E3,E4,E5,E6,E7,E8 emb;
```

The 9 blue-grey boxes are **embedded messages**, not resources. They have no
name of their own and travel with the adas.

## Example

A adas as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/adas",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isAutoPowerOptimize": true,
  "powerOptimizeLevel": 3,
  "activeAutonomyLevel": "ACTIVE_AUTONOMY_LEVEL_SAE_1",
  "supportedAutonomyLevel": "SUPPORTED_AUTONOMY_LEVEL_SAE_1"
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.ADAS.IsAutoPowerOptimize": true,
  "Vehicle.ADAS.PowerOptimizeLevel": 3,
  "Vehicle.ADAS.ActiveAutonomyLevel": "SAE_1",
  "Vehicle.ADAS.SupportedAutonomyLevel": "SAE_1"
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

Writing to a adas, and the one thing that will fail:

```mermaid
sequenceDiagram
  participant C as Client
  participant A as AdasService
  participant V as Vehicle

  Note over C,V: is_auto_power_optimize is an actuator — the API is the source
  C->>A: PATCH …?updateMask=isAutoPowerOptimize
  A->>V: set is_auto_power_optimize
  V-->>A: acknowledged
  A-->>C: 200 OK

  Note over C,V: active_autonomy_level is a sensor — the vehicle is the source
  C->>A: PATCH …?updateMask=activeAutonomyLevel
  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field
```

```http
PATCH /v1/vehicles/wvwzzz1jz3w000001/adas?updateMask=isAutoPowerOptimize
{ "isAutoPowerOptimize": true }
```

The rejection is the schema working, not an inconvenience: VSS marks
`active_autonomy_level` a sensor, so the vehicle is the only thing that can
set it. A mask naming it fails rather than being silently dropped, so a caller
that believed it had written the value finds out immediately.

## Resource names

Every adas is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-30: "adas"
```

31 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/adas")
t.Parse("vehicles/wvwzzz1jz3w000001/adas")
// map[vehicle:wvwzzz1jz3w000001]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/adas")
t.parse("vehicles/wvwzzz1jz3w000001/adas")
# {'vehicle': 'wvwzzz1jz3w000001'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

19 fields: 7 AIP identity and lifecycle, 12 VSS signals. Field numbers 8–15 are reserved for identity fields a later revision may add, so adding one never renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `abs` | `Abs` | — | `Vehicle.ADAS.ABS` | Antilock Braking System signals. |
| `cruise_control` | `CruiseControl` | — | `Vehicle.ADAS.CruiseControl` | Signals from Cruise Control system. |
| `dms` | `Dms` | — | `Vehicle.ADAS.DMS` | Driver Monitoring System signals. |
| `eba` | `Eba` | — | `Vehicle.ADAS.EBA` | Emergency Brake Assist (EBA) System signals. |
| `ebd` | `Ebd` | — | `Vehicle.ADAS.EBD` | Electronic Brakeforce Distribution (EBD) System signals. |
| `esc` | `Esc` | — | `Vehicle.ADAS.ESC` | Electronic Stability Control System signals. |
| `is_auto_power_optimize` | `bool` | — | `Vehicle.ADAS.IsAutoPowerOptimize` | Auto Power Optimization Flag When set to 'true', the system enables automatic power optimization, dynamically adjusting the power optimization level based on runtime conditions or features managed by the OEM. When set to 'false', manual control of the power optimization level is allowed. |
| `lane_departure_detection` | `LaneDepartureDetection` | — | `Vehicle.ADAS.LaneDepartureDetection` | Signals from Lane Departure Detection System. |
| `power_optimize_level` | `int32` | — | `Vehicle.ADAS.PowerOptimizeLevel` | Power optimization level for this branch/subsystem. A higher number indicates more aggressive power optimization. Level 0 indicates that all functionality is enabled, no power optimization enabled. Level 10 indicates most aggressive power optimization mode, only essential functionality enabled. |
| `tcs` | `Tcs` | — | `Vehicle.ADAS.TCS` | Traction Control System signals. |

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `active_autonomy_level` | [`ActiveAutonomyLevel`](#activeautonomylevel) | — | `Vehicle.ADAS.ActiveAutonomyLevel` | Indicates the currently active level of driving automation according to the SAE J3016 (Taxonomy and Definitions for Terms Related to Driving Automation Systems for On-Road Motor Vehicles). |
| `supported_autonomy_level` | [`SupportedAutonomyLevel`](#supportedautonomylevel) | — | `Vehicle.ADAS.SupportedAutonomyLevel` | Indicates the highest level of driving automation according to the SAE J3016 taxonomy the vehicle is capable of. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/adas`, server-assigned |
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
GET    /v1/{name=vehicles/*/adas}
PATCH  /v1/{adas.name=vehicles/*/adas}
```

## Enums

#### `ActiveAutonomyLevel`

Indicates the currently active level of driving automation according to the
SAE J3016 (Taxonomy and Definitions for Terms Related to Driving Automation
Systems for On-Road Motor Vehicles).

`UNSPECIFIED` · `SAE_0` · `SAE_1` · `SAE_2_DISENGAGING` · `SAE_2` ·
`SAE_3_DISENGAGING` · `SAE_3` · `SAE_4_DISENGAGING` · `SAE_4` ·
`SAE_5_DISENGAGING` · `SAE_5`
<sub>emitted as `ACTIVE_AUTONOMY_LEVEL_SAE_0`, and so on</sub>

#### `SupportedAutonomyLevel`

Indicates the highest level of driving automation according to the SAE J3016
taxonomy the vehicle is capable of.

`UNSPECIFIED` · `SAE_0` · `SAE_1` · `SAE_2` · `SAE_3` · `SAE_4` ·
`SAE_5`
<sub>emitted as `SUPPORTED_AUTONOMY_LEVEL_SAE_0`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`adas.proto`](v1/adas.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
