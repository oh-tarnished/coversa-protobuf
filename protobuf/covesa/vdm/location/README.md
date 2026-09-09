# Location

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-ChargingStation.Location-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-singleton-2B3172)](https://aip.dev/156)
[![RPCs](https://img.shields.io/badge/RPCs-2-555)](#methods)
[![signals](https://img.shields.io/badge/signals-5-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vdm.location.v1-444)](v1/)

Location is a node of the COVESA Vehicle Signal Specification.

```
chargingStations/{charging_station}/location
```

## Where it sits

```mermaid
flowchart LR
  P["ChargingStation<br/><code>chargingStations/{charging_station}</code>"]
  R["Location<br/><code>…/{charging_station}/location</code>"]

  P -->|"exactly one"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

## Example

A location as this API returns it:

```json
{
  "name": "chargingStations/{charging_station}/location",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "administrativeArea": "<administrative area>",
  "city": "<city>",
  "regionCode": "SE",
  "street": "<street>",
  "zipCode": "<zip code>"
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "ChargingStation.Location.State": "<administrative area>",
  "ChargingStation.Location.City": "<city>",
  "ChargingStation.Location.Country": "SE",
  "ChargingStation.Location.Street": "<street>",
  "ChargingStation.Location.ZipCode": "<zip code>"
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every location is addressed by a name that alternates collection and
identifier, per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-15: "chargingStations"
16: "/"
17-34: "{charging_station}"
35: "/"
36-43: "location"
```

44 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("chargingStations/{charging_station}/location")
t.Parse("chargingStations/{charging_station}/location")
// map[chargingStation:chargingStation-yqey58b3bwr1x80j]
```

```python
t = resourcename.ResourceTemplate("chargingStations/{charging_station}/location")
t.parse("chargingStations/{charging_station}/location")
# {'chargingStation': 'chargingStation-yqey58b3bwr1x80j'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

12 fields: 7 AIP identity and lifecycle, 5 VSS signals. Field numbers 8–15
are reserved for identity fields a later revision may add, so adding one never
renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `administrative_area` | `string` | — | `ChargingStation.Location.State` | Administrative area. |
| `city` | `string` | — | `ChargingStation.Location.City` | City. |
| `region_code` | `string` | — | `ChargingStation.Location.Country` | Region code. |
| `street` | `string` | — | `ChargingStation.Location.Street` | Street. |
| `zip_code` | `string` | — | `ChargingStation.Location.ZipCode` | Zip code. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `chargingStations/{charging_station}/location`, server-assigned |
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
GET    /v1/{name=chargingStations/*/location}
PATCH  /v1/{location.name=chargingStations/*/location}
```

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`location.proto`](v1/location.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
