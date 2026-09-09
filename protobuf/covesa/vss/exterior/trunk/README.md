# Trunk

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Body.Trunk-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-5-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.exterior.trunk.v1-444)](v1/)

Trunk status. Start position for Trunk is Closed.

```
vehicles/{vehicle}/trunks/{trunk}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Trunk<br/><code>…/trunks/{trunk}</code>"]

  P -->|"many, each identified"| R

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
```

In the specification this hangs beneath **Body**, but its name hangs off
**Vehicle**. AIP-123 requires a name to alternate collection and identifier,
and `body` is a singleton whose name ends in a literal — two literals in a
row is not a legal name. Nothing is lost: body has one occurrence, so the
segment would identify nothing, and the full VSS path survives in the branch
annotation.

## Example

A trunk as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/trunks/trunk-z15hvn7jrc27a5ey",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isLightOn": true,
  "isLocked": true,
  "isOpen": true,
  "position": 33,
  "switchControl": "SWITCH_CLOSE"
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Body.Trunk.IsLightOn": true,
  "Vehicle.Body.Trunk.IsLocked": true,
  "Vehicle.Body.Trunk.IsOpen": true,
  "Vehicle.Body.Trunk.Position": 33,
  "Vehicle.Body.Trunk.Switch": "CLOSE"
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

## Resource names

Every trunk is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-32: "trunks"
33: "/"
34-55: "trunk-z15hvn7jrc27a5ey"
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/trunks/{trunk}")
t.Parse("vehicles/wvwzzz1jz3w000001/trunks/trunk-z15hvn7jrc27a5ey")
// map[vehicle:wvwzzz1jz3w000001 trunk:trunk-z15hvn7jrc27a5ey]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/trunks/{trunk}")
t.parse("vehicles/wvwzzz1jz3w000001/trunks/trunk-z15hvn7jrc27a5ey")
# {'vehicle': 'wvwzzz1jz3w000001', 'trunk': 'trunk-z15hvn7jrc27a5ey'}
```

The template above is the `pattern` this resource declares in its
`google.api.resource` annotation, so the two cannot drift.

## Signals

12 fields: 7 AIP identity and lifecycle, 5 VSS signals. Field numbers 8–15 are reserved for identity fields a later revision may add, so adding one never renumbers a signal.

### Writable — actuators

The vehicle accepts these in an `update_mask`.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_light_on` | `bool` | — | `Vehicle.Body.Trunk.IsLightOn` | Is trunk light on |
| `is_locked` | `bool` | — | `Vehicle.Body.Trunk.IsLocked` | Is item locked or unlocked. True = Locked. False = Unlocked. |
| `is_open` | `bool` | — | `Vehicle.Body.Trunk.IsOpen` | Is item open or closed? True = Fully or partially open. False = Fully closed. |
| `position` | `int32` | `percent` | `Vehicle.Body.Trunk.Position` | Item position. 0 = Start position 100 = End position. |
| `switch_control` | [`Switch`](#switch) | — | `Vehicle.Body.Trunk.Switch` | Switch controlling sliding action such as window, sunroof, or blind. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/trunks/{trunk}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Which trunk

```mermaid
flowchart LR
  A0["Front"]
  A1["Rear"]
```

VSS expands this branch across Front, Rear, so the combination names one trunk
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
GET    /v1/{name=vehicles/*/trunks/*}
PATCH  /v1/{trunk.name=vehicles/*/trunks/*}
GET    /v1/{parent=vehicles/*}/trunks
POST   /v1/{parent=vehicles/*}/trunks
DELETE /v1/{name=vehicles/*/trunks/*}
POST   /v1/{name=vehicles/*/trunks/*}:undelete
```

A deleted trunk is still returned by `Get` and hidden from `List` unless
`show_deleted` is set — which is what makes `Undelete` meaningful.

## Enums

#### `Switch`

Switch controlling sliding action such as window, sunroof, or blind.

`UNSPECIFIED` · `INACTIVE` · `CLOSE` · `OPEN` · `ONE_SHOT_CLOSE` ·
`ONE_SHOT_OPEN`
<sub>emitted as `SWITCH_INACTIVE`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`trunk.proto`](v1/trunk.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
