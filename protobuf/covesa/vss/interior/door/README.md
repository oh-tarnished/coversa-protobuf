# Door

[![VSS](https://img.shields.io/badge/VSS-2026--09--02-0B6B5B)](https://github.com/COVESA/vehicle_signal_specification)
[![branch](https://img.shields.io/badge/branch-Vehicle.Cabin.Door-1D4ED8)](https://covesa.github.io/vehicle_signal_specification/)
[![shape](https://img.shields.io/badge/shape-collection-2B3172)](https://aip.dev/121)
[![RPCs](https://img.shields.io/badge/RPCs-6-555)](#methods)
[![signals](https://img.shields.io/badge/signals-5-7A4A00)](#signals)
[![package](https://img.shields.io/badge/package-protobuf.covesa.vss.interior.door.v1-444)](v1/)

All doors, including windows and switches.

```
vehicles/{vehicle}/doors/{door}
```

## Where it sits

```mermaid
flowchart LR
  P["Vehicle<br/><code>vehicles/{vehicle}</code>"]
  R["Door<br/><code>…/doors/{door}</code>"]
  E0["Window"]
  E1["Shade"]

  P -->|"many, each identified"| R
  R --- E0
  R --- E1

  classDef res fill:#2B3172,stroke:#6C74C4,color:#fff;
  classDef emb fill:#1d2440,stroke:#4a5178,color:#cfd4ea;
  class R,P res;
  class E0,E1 emb;
```

In the specification this hangs beneath **Cabin**, but its name hangs off
**Vehicle**. AIP-123 requires a name to alternate collection and identifier,
and `cabin` is a singleton whose name ends in a literal — two literals in a
row is not a legal name. Nothing is lost: cabin has one occurrence, so the
segment would identify nothing, and the full VSS path survives in the branch
annotation.

The 2 blue-grey boxes are **embedded messages**, not resources. They have no
name of their own and travel with the door.

## Example

A door as this API returns it:

```json
{
  "name": "vehicles/wvwzzz1jz3w000001/doors/door-6xqvf9f63vr5p2y1",
  "uid": "b3f1c2de-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
  "isLocked": true,
  "isOpen": true,
  "position": 33,
  "switchControl": "DOOR_SWITCH_CLOSE",
  "isChildLockActive": true
}
```

The same data in the **VSS shape**, for a peer that speaks the specification
rather than this schema. `codec.ToVSS` produces it from
[`codec/manifest.json`](../../../../../codec/manifest.json):

```json
{
  "Vehicle.Cabin.Door.IsLocked": true,
  "Vehicle.Cabin.Door.IsOpen": true,
  "Vehicle.Cabin.Door.Position": 33,
  "Vehicle.Cabin.Door.Switch": "CLOSE",
  "Vehicle.Cabin.Door.IsChildLockActive": true
}
```

Note what changed: signals are keyed by **fully qualified VSS name**, enum
constants drop to the spelling VSS writes, and the AIP fields — `name`,
`uid` — fall away, because VSS declares no signal for them. `codec.FromVSS`
reverses all three.

Writing to a door, and the one thing that will fail:

```mermaid
sequenceDiagram
  participant C as Client
  participant A as Doors
  participant V as Vehicle

  Note over C,V: is_locked is an actuator — the API is the source
  C->>A: PATCH …?updateMask=isLocked
  A->>V: set is_locked
  V-->>A: acknowledged
  A-->>C: 200 OK

  Note over C,V: is_child_lock_active is a sensor — the vehicle is the source
  C->>A: PATCH …?updateMask=isChildLockActive
  A--xC: 400 INVALID_ARGUMENT — update_mask names a read-only field
```

```http
PATCH /v1/vehicles/wvwzzz1jz3w000001/doors/door-6xqvf9f63vr5p2y1?updateMask=isLocked
{ "isLocked": true }
```

The rejection is the schema working, not an inconvenience: VSS marks
`is_child_lock_active` a sensor, so the vehicle is the only thing that can set
it. A mask naming it fails rather than being silently dropped, so a caller
that believed it had written the value finds out immediately.

## Resource names

Every door is addressed by a name that alternates collection and identifier,
per [AIP-122](https://aip.dev/122):

```mermaid
packet
0-7: "vehicles"
8: "/"
9-25: "wvwzzz1jz3w000001"
26: "/"
27-31: "doors"
32: "/"
33-53: "door-6xqvf9f63vr5p2y1"
```

54 characters, alternating, which is what [AIP-123](https://aip.dev/123)
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
t := resourcename.ResourceTemplate("vehicles/{vehicle}/doors/{door}")
t.Parse("vehicles/wvwzzz1jz3w000001/doors/door-6xqvf9f63vr5p2y1")
// map[vehicle:wvwzzz1jz3w000001 door:door-6xqvf9f63vr5p2y1]
```

```python
t = resourcename.ResourceTemplate("vehicles/{vehicle}/doors/{door}")
t.parse("vehicles/wvwzzz1jz3w000001/doors/door-6xqvf9f63vr5p2y1")
# {'vehicle': 'wvwzzz1jz3w000001', 'door': 'door-6xqvf9f63vr5p2y1'}
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
| `is_locked` | `bool` | — | `Vehicle.Cabin.Door.IsLocked` | Is item locked or unlocked. True = Locked. False = Unlocked. |
| `is_open` | `bool` | — | `Vehicle.Cabin.Door.IsOpen` | Is item open or closed? True = Fully or partially open. False = Fully closed. |
| `position` | `int32` | `percent` | `Vehicle.Cabin.Door.Position` | Item position. 0 = Start position 100 = End position. |
| `switch_control` | [`DoorSwitch`](#doorswitch) | — | `Vehicle.Cabin.Door.Switch` | Switch controlling sliding action such as window, sunroof, or blind. |

### Read-only — sensors and attributes

The vehicle reports these. An `update_mask` naming one is **rejected**, not
silently ignored.

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_child_lock_active` | `bool` | — | `Vehicle.Cabin.Door.IsChildLockActive` | Is door child lock active. True = Door cannot be opened from inside. False = Door can be opened from inside. |

<details>
<summary>Identity and lifecycle — 7 fields every resource carries</summary>

| Field | Type | Notes |
| --- | --- | --- |
| `name` | `string` | `vehicles/{vehicle}/doors/{door}`, server-assigned |
| `uid` | `string` | server-assigned UUID4, per [AIP-148](https://aip.dev/148) |
| `etag` | `string` | pass back on update to make the write conditional, per [AIP-154](https://aip.dev/154) |
| `create_time` `update_time` | `Timestamp` | when it was stored here |
| `delete_time` `expire_time` | `Timestamp` | soft delete; recoverable until `expire_time` |

</details>

## Embedded messages

2 messages travel inside the door and have no name of their own. Address a
field on one through its owner — `window.<field>` — not directly.

<details>
<summary><code>Window</code> — Door window status. Start position for Window is Closed.</summary>

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_open` | `bool` | — | `Vehicle.Cabin.Door.Window.IsOpen` | Is item open or closed? True = Fully or partially open. False = Fully closed. |
| `position` | `int32` | `percent` | `Vehicle.Cabin.Door.Window.Position` | Item position. 0 = Start position 100 = End position. |
| `switch_control` | [`WindowSwitch`](#windowswitch) | — | `Vehicle.Cabin.Door.Window.Switch` | Switch controlling sliding action such as window, sunroof, or blind. |

</details>

<details>
<summary><code>Shade</code> — Side window shade. Open = Retracted, Closed = Deployed. Start position for Shade is Open/Retracted.</summary>

| Field | Type | Unit | VSS | Description |
| --- | --- | --- | --- | --- |
| `is_open` | `bool` | — | `Vehicle.Cabin.Door.Shade.IsOpen` | Is item open or closed? True = Fully or partially open. False = Fully closed. |
| `position` | `int32` | `percent` | `Vehicle.Cabin.Door.Shade.Position` | Item position. 0 = Start position 100 = End position. |
| `switch_control` | [`ShadeSwitch`](#shadeswitch) | — | `Vehicle.Cabin.Door.Shade.Switch` | Switch controlling sliding action such as window, sunroof, or blind. |

</details>

## Which door

```mermaid
flowchart LR
  O0["Row1"]
  O0 --> I0_0["DriverSide"]
  O0 --> I0_1["PassengerSide"]
  O1["Row2"]
  O1 --> I1_0["DriverSide"]
  O1 --> I1_1["PassengerSide"]
```

VSS expands this branch across Row1, Row2 × DriverSide, PassengerSide, so the
combination names one door — which is what makes it a resource rather than a
repeated field. The axes are recorded in
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
GET    /v1/{name=vehicles/*/doors/*}
PATCH  /v1/{door.name=vehicles/*/doors/*}
GET    /v1/{parent=vehicles/*}/doors
POST   /v1/{parent=vehicles/*}/doors
DELETE /v1/{name=vehicles/*/doors/*}
POST   /v1/{name=vehicles/*/doors/*}:undelete
```

A deleted door is still returned by `Get` and hidden from `List` unless
`show_deleted` is set — which is what makes `Undelete` meaningful.

## Enums

#### `DoorSwitch`

Switch controlling sliding action such as window, sunroof, or blind.

`UNSPECIFIED` · `INACTIVE` · `CLOSE` · `OPEN` · `ONE_SHOT_CLOSE` ·
`ONE_SHOT_OPEN`
<sub>emitted as `DOOR_SWITCH_INACTIVE`, and so on</sub>

#### `WindowSwitch`

Switch controlling sliding action such as window, sunroof, or blind.

`UNSPECIFIED` · `INACTIVE` · `CLOSE` · `OPEN` · `ONE_SHOT_CLOSE` ·
`ONE_SHOT_OPEN`
<sub>emitted as `WINDOW_SWITCH_INACTIVE`, and so on</sub>

#### `ShadeSwitch`

Switch controlling sliding action such as window, sunroof, or blind.

`UNSPECIFIED` · `INACTIVE` · `CLOSE` · `OPEN` · `ONE_SHOT_CLOSE` ·
`ONE_SHOT_OPEN`
<sub>emitted as `SHADE_SWITCH_INACTIVE`, and so on</sub>

---

<sub>Generated by `just docs` from COVESA VSS `2026-09-02` (`cd4bc50`) and VDM `2026-07-24` (`36bc939`). Do not edit by hand.<br>
Source: [`door.proto`](v1/door.proto) · [`service.proto`](v1/service.proto) · [`messages.proto`](v1/messages.proto)</sub>
