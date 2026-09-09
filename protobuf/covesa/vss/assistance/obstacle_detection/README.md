# ObstacleDetection

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.ADAS.ObstacleDetection-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-6-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.assistance.obstacle__detection.v1-444)](v1/)

Signals form Obstacle Sensor System.

```
vehicles/{vehicle}/obstacleDetections/{obstacle_detection}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["ObstacleDetection<br/><code>…/obstacleDetections/{obstacle_detection}</code>"]

  P -->|"many, each identified"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

In the specification this hangs beneath **Adas**, but its name hangs off
**Vehicle**. AIP-123 requires a name to alternate collection and identifier,
and `adas` is a singleton whose name ends in a literal — two literals in a
row is not a legal name. Nothing is lost: adas has one occurrence, so the
segment would identify nothing, and the full VSS path survives in the branch
annotation.

## Example

A obstacleDetection as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/obstacleDetections/{obstacle_detection}",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isEnabled": true,
  "distance": 88.5,
  "isError": true,
  "isWarning": true,
  "timeGap": 42
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.ADAS.ObstacleDetection.IsEnabled": true,
  "Vehicle.ADAS.ObstacleDetection.Distance": 88.5,
  "Vehicle.ADAS.ObstacleDetection.IsError": true,
  "Vehicle.ADAS.ObstacleDetection.IsWarning": true,
  "Vehicle.ADAS.ObstacleDetection.TimeGap": 42
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

Writing to a obstacleDetection, and the one thing that will fail:

```mermaid
sequenceDiagram
  participant C as Client
  participant A as ObstacleDetections
  participant V as Vehicle

  Note over C,V: is_enabled is an actuator — the API is the source
  C->>A: PATCH …?updateMask=isEnabled
  A->>V: set is_enabled
  V-->>A: acknowledged
  A-->>C: 200 OK

  Note over C,V: distance is a sensor — the vehicle is the source
  C->>A: PATCH …?updateMask=distance
  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field
```

```http
PATCH /v1/vehicles/wvwzzz1jz3w000001/obstacleDetections/{obstacle_detection}?updateMask=isEnabled
{ "isEnabled": true }
```

The rejection is the schema working, not an inconvenience: VSS marks
`distance` a sensor, so the vehicle is the only thing that can set it. A mask
naming it fails rather than being silently dropped, so a caller that believed
it had written the value finds out immediately.

## Resource names

Every obstacleDetection is addressed by a name that alternates collection and
identifier, per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-44: "obstacleDetections"
45: "/"
46-65: "{obstacle_detection}"
```

66 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/obstacleDetections/{obstacle_detection}")
t.Parse("vehicles/wvwzzz1jz3w000001/obstacleDetections/{obstacle_detection}")
// map[vehicle:wvwzzz1jz3w000001 obstacleDetection:obstacleDetection-wjwcssz1hyt8amb3]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/obstacleDetections/{obstacle_detection}")
t.parse("vehicles/wvwzzz1jz3w000001/obstacleDetections/{obstacle_detection}")
# {'vehicle': 'wvwzzz1jz3w000001', 'obstacleDetection': 'obstacleDetection-wjwcssz1hyt8amb3'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

13 fields: 7 AIP identity and lifecycle, 6 VSS signals. Field numbers 8–15 are reserved for identity fields a later revision may add, so adding one never renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_enabled` | `bool` | — | `Vehicle.ADAS.ObstacleDetection.IsEnabled` | Indicates if obstacle sensor system is enabled (i.e. monitoring for obstacles). True = Enabled. False = Disabled. |

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `distance` | `double` | `m` | `Vehicle.ADAS.ObstacleDetection.Distance` | Distance in meters to detected object |
| `is_error` | `bool` | — | `Vehicle.ADAS.ObstacleDetection.IsError` | Indicates if obstacle sensor system incurred an error condition. True = Error. False = No Error. |
| `is_warning` | `bool` | — | `Vehicle.ADAS.ObstacleDetection.IsWarning` | Indicates if obstacle sensor system registered an obstacle. |
| `time_gap` | `int64` | `ms` | `Vehicle.ADAS.ObstacleDetection.TimeGap` | Time in milliseconds before potential impact object |
| `warning_type` | [`WarningType`](#warningtype) | — | `Vehicle.ADAS.ObstacleDetection.WarningType` | Indicates the type of obstacle warning detected as some track not only the presence of an obstacle but potential intercepting trajectory or other characteristics. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/obstacleDetections/{obstacle_detection}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which obstacleDetection

```mermaid
flowchart LR
  O0["Front"]
  O0 --> I0_0["Left"]
  O0 --> I0_1["Center"]
  O0 --> I0_2["Right"]
  O1["Rear"]
  O1 --> I1_0["Left"]
  O1 --> I1_1["Center"]
  O1 --> I1_2["Right"]
```

VSS expands this branch across Front, Rear × Left, Center, Right, so the
combination names one obstacleDetection — which is what makes it a resource
rather than a repeated field. The axes are recorded in
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
GET    /v1/{name=vehicles/*/obstacleDetections/*}
PATCH  /v1/{obstacleDetection.name=vehicles/*/obstacleDetections/*}
GET    /v1/{parent=vehicles/*}/obstacleDetections
POST   /v1/{parent=vehicles/*}/obstacleDetections
DELETE /v1/{name=vehicles/*/obstacleDetections/*}
POST   /v1/{name=vehicles/*/obstacleDetections/*}:undelete
```

A deleted obstacleDetection is still returned by `Get` and hidden from `List`
unless `show_deleted` is set — which is what makes `Undelete` meaningful.

## Enums

#### `WarningType`

Indicates the type of obstacle warning detected as some track not only the
presence of an obstacle but potential intercepting trajectory or other
characteristics.

`UNSPECIFIED` · `UNDEFINED` · `CROSS_TRAFFIC` · `BLIND_SPOT`
<sub>emitted as `WARNING_TYPE_UNDEFINED`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`obstacle_detection.proto`](v1/obstacle_detection.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
