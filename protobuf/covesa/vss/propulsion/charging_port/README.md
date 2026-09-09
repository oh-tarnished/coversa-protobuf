# ChargingPort

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Powertrain.TractionBattery.Charging.ChargingPort-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-4-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.propulsion.charging__port.v1-444)](v1/)

Properties related to a particular charging port available in the vehicle.

```
vehicles/{vehicle}/chargingPorts/{charging_port}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["ChargingPort<br/><code>…/chargingPorts/{charging_port}</code>"]

  P -->|"many, each identified"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

In the specification this hangs beneath **Powertrain**, but its name hangs off
**Vehicle**. AIP-123 requires a name to alternate collection and identifier,
and `powertrain` is a singleton whose name ends in a literal — two literals
in a row is not a legal name. Nothing is lost: powertrain has one occurrence,
so the segment would identify nothing, and the full VSS path survives in the
branch annotation.

## Example

A chargingPort as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/chargingPorts/{charging_port}",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isChargingCableLocked": true,
  "isFlapOpen": true,
  "isChargingCableConnected": true
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsChargingCableLocked": true,
  "Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsFlapOpen": true,
  "Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsChargingCableConnected": true
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

Writing to a chargingPort, and the one thing that will fail:

```mermaid
sequenceDiagram
  participant C as Client
  participant A as ChargingPorts
  participant V as Vehicle

  Note over C,V: is_charging_cable_locked is an actuator — the API is the source
  C->>A: PATCH …?updateMask=isChargingCableLocked
  A->>V: set is_charging_cable_locked
  V-->>A: acknowledged
  A-->>C: 200 OK

  Note over C,V: is_charging_cable_connected is a sensor — the vehicle is the source
  C->>A: PATCH …?updateMask=isChargingCableConnected
  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field
```

```http
PATCH /v1/vehicles/wvwzzz1jz3w000001/chargingPorts/{charging_port}?updateMask=isChargingCableLocked
{ "isChargingCableLocked": true }
```

The rejection is the schema working, not an inconvenience: VSS marks
`is_charging_cable_connected` a sensor, so the vehicle is the only thing that
can set it. A mask naming it fails rather than being silently dropped, so a
caller that believed it had written the value finds out immediately.

## Resource names

Every chargingPort is addressed by a name that alternates collection and
identifier, per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-39: "chargingPorts"
40: "/"
41-55: "{charging_port}"
```

56 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/chargingPorts/{charging_port}")
t.Parse("vehicles/wvwzzz1jz3w000001/chargingPorts/{charging_port}")
// map[vehicle:wvwzzz1jz3w000001 chargingPort:chargingPort-gthkwmvp50nvx0xs]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/chargingPorts/{charging_port}")
t.parse("vehicles/wvwzzz1jz3w000001/chargingPorts/{charging_port}")
# {'vehicle': 'wvwzzz1jz3w000001', 'chargingPort': 'chargingPort-gthkwmvp50nvx0xs'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

11 fields: 7 AIP identity and lifecycle, 4 VSS signals. Field numbers 8–15
are reserved for identity fields a later revision may add, so adding one never
renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_charging_cable_locked` | `bool` | — | `Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsChargingCableLocked` | Is charging cable locked to prevent removal. |
| `is_flap_open` | `bool` | — | `Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsFlapOpen` | Status of the charging port flap(s). |

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_charging_cable_connected` | `bool` | — | `Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.IsChargingCableConnected` | Indicates whether a charging cable is physically connected to a particular charging port or not. |
| `supported_inlet_types` | [`repeated SupportedInletTypes`](#repeated supportedinlettypes) | — | `Vehicle.Powertrain.TractionBattery.Charging.ChargingPort.SupportedInletTypes` | A list of the supported (i.e., available) charging inlets in a particular charging port. IEC types refer to IEC 62196, GBT refers to GB/T 20234. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/chargingPorts/{charging_port}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which chargingPort

```mermaid
flowchart LR
  A0["FrontLeft"]
  A1["FrontMiddle"]
  A2["FrontRight"]
  A3["RearLeft"]
  A4["RearMiddle"]
  A5["RearRight"]
  A6["AnyPosition"]
```

VSS expands this branch across FrontLeft, FrontMiddle, FrontRight, RearLeft,
RearMiddle, RearRight, AnyPosition, so the combination names one chargingPort
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
GET    /v1/{name=vehicles/*/chargingPorts/*}
PATCH  /v1/{chargingPort.name=vehicles/*/chargingPorts/*}
GET    /v1/{parent=vehicles/*}/chargingPorts
POST   /v1/{parent=vehicles/*}/chargingPorts
DELETE /v1/{name=vehicles/*/chargingPorts/*}
POST   /v1/{name=vehicles/*/chargingPorts/*}:undelete
```

A deleted chargingPort is still returned by `Get` and hidden from `List`
unless `show_deleted` is set — which is what makes `Undelete` meaningful.

## Enums

#### `SupportedInletTypes`

A list of the supported (i.e., available) charging inlets in a particular
charging port. IEC types refer to IEC 62196, GBT refers to GB/T 20234.

`UNSPECIFIED` · `IEC_TYPE_1_AC` · `IEC_TYPE_2_AC` · `IEC_TYPE_3_AC` ·
`IEC_TYPE_4_DC` · `IEC_TYPE_1_CCS_DC` · `IEC_TYPE_2_CCS_DC` ·
`TESLA_ROADSTER` · `TESLA_HPWC` · `TESLA_SUPERCHARGER` · `GBT_AC` ·
`GBT_DC` · `OTHER`
<sub>emitted as `SUPPORTED_INLET_TYPES_IEC_TYPE_1_AC`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`charging_port.proto`](v1/charging_port.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
