# Station

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Cabin.HVAC.Station-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-3-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.interior.station.v1-444)](v1/)

HVAC for single station in the vehicle

```
vehicles/{vehicle}/stations/{station}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Station<br/><code>…/stations/{station}</code>"]

  P -->|"many, each identified"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

In the specification this hangs beneath **Cabin**, but its name hangs off
**Vehicle**. AIP-123 requires a name to alternate collection and identifier,
and `cabin` is a singleton whose name ends in a literal — two literals in a
row is not a legal name. Nothing is lost: cabin has one occurrence, so the
segment would identify nothing, and the full VSS path survives in the branch
annotation.

## Example

A station as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/stations/station-6m9b4tz33pfm59g9",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "airDistribution": "AIR_DISTRIBUTION_MIDDLE",
  "fanSpeed": 33,
  "temperature": 88.5
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Cabin.HVAC.Station.AirDistribution": "MIDDLE",
  "Vehicle.Cabin.HVAC.Station.FanSpeed": 33,
  "Vehicle.Cabin.HVAC.Station.Temperature": 88.5
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every station is addressed by a name that alternates collection and
identifier, per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-34: "stations"
35: "/"
36-59: "station-6m9b4tz33pfm59g9"
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/stations/{station}")
t.Parse("vehicles/wvwzzz1jz3w000001/stations/station-6m9b4tz33pfm59g9")
// map[vehicle:wvwzzz1jz3w000001 station:station-6m9b4tz33pfm59g9]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/stations/{station}")
t.parse("vehicles/wvwzzz1jz3w000001/stations/station-6m9b4tz33pfm59g9")
# {'vehicle': 'wvwzzz1jz3w000001', 'station': 'station-6m9b4tz33pfm59g9'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

10 fields: 7 AIP identity and lifecycle, 3 VSS signals. Field numbers 8–15
are reserved for identity fields a later revision may add, so adding one never
renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `air_distribution` | [`AirDistribution`](#airdistribution) | — | `Vehicle.Cabin.HVAC.Station.AirDistribution` | Direction of airstream |
| `fan_speed` | `int32` | `percent` | `Vehicle.Cabin.HVAC.Station.FanSpeed` | Fan Speed, 0 = off. 100 = max |
| `temperature` | `double` | `Celsius` | `Vehicle.Cabin.HVAC.Station.Temperature` | Temperature |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/stations/{station}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which station

```mermaid
flowchart LR
  O0["Row1"]
  O0 --> I0_0["Driver"]
  O0 --> I0_1["Passenger"]
  O1["Row2"]
  O1 --> I1_0["Driver"]
  O1 --> I1_1["Passenger"]
  O2["Row3"]
  O2 --> I2_0["Driver"]
  O2 --> I2_1["Passenger"]
  O3["Row4"]
  O3 --> I3_0["Driver"]
  O3 --> I3_1["Passenger"]
```

VSS expands this branch across Row1, Row2, Row3, Row4 × Driver, Passenger, so
the combination names one station — which is what makes it a resource rather
than a repeated field. The axes are recorded in
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
GET    /v1/{name=vehicles/*/stations/*}
PATCH  /v1/{station.name=vehicles/*/stations/*}
GET    /v1/{parent=vehicles/*}/stations
POST   /v1/{parent=vehicles/*}/stations
DELETE /v1/{name=vehicles/*/stations/*}
POST   /v1/{name=vehicles/*/stations/*}:undelete
```

A deleted station is still returned by `Get` and hidden from `List` unless
`show_deleted` is set — which is what makes `Undelete` meaningful.

## Enums

#### `AirDistribution`

Direction of airstream

`UNSPECIFIED` · `UP` · `MIDDLE` · `DOWN`
<sub>emitted as `AIR_DISTRIBUTION_UP`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`station.proto`](v1/station.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
