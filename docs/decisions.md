# Decisions

What was deliberately not done, and why. Each of these was tried, or was the
obvious choice and rejected for a reason worth recording.

## The vehicle tree is many packages, not one message

The source model gives `Vehicle` 28 branch fields and reaches 278 types. The
faithful protobuf translation is one enormous message with `Cabin`,
`Powertrain` and the rest embedded, and it was rejected: AIP-215 would allow
it only inside a single package, so reading a vehicle's cabin would transfer
its powertrain, and nothing could address a seat.

Instead each branch is a resource. The cost is real and is stated in every
service comment: **the tree is navigated by resource name, not read whole in
one call.** Reading a vehicle and its cabin is two requests and they are not
atomic. That is the price of being able to read a cabin without a powertrain.

## Instance-tagged branches are resources, at any depth

22 branches carry `@instanceTag` in the source model -- seats, doors, wheels,
mirrors, charging ports. That annotation is the specification stating that
the members of a repeated branch have identity: a seat is Row1/DriverSide.

They were embedded messages in the first cut, which meant `PATCH` on a single
seat was impossible; you replaced the whole cabin. Promoting them made 22 new
resources and the schema markedly more useful. A grouping node with one
occurrence is *not* promoted -- `Cabin.Light` stays inside the cabin -- because
a segment with one possible value identifies nothing.

## Units are pinned, not carried

Rejected: a `unit` field beside every measurement, and a `{value, unit}`
wrapper message.

The field lets two producers of `speed` disagree about what the number means,
which is precisely the failure the source model prevents by giving the
argument a default. The wrapper turns every numeric signal into a submessage,
which defeats the zero-copy read the Cap'n Proto layer exists for.

So the schema pins the source default and records it in an annotation. See
rule 12.

## `google.type.Date` and `google.type.PostalAddress` were withdrawn

Both were adopted first, and they are the *correct* protobuf answer: AIP-215
exempts `google.*` precisely so a value type can be shared across packages,
and `PostalAddress` models the source model's `Address` without loss.

They were withdrawn because `protoc-gen-buffers` does not render them.
`Person.home_address` **vanished from the emitted FlatBuffers table with no
diagnostic** -- 19 fields in the proto, 18 in the `.fbs` -- and
`google.type.Date` produced an `include` of a file the generator never wrote,
so five schemas did not compile at all.

A schema that quietly means something different in one of its own target
formats is worse than a duplicated message. `Address` and `Date` are now
generated into each package that needs them. The duplication is real; it is
generated rather than maintained, and it is visible.

This is the one place these conventions diverge from protobuf-rfc's rule 4.

## `CountryCode` is a string, not an enum

The source model enumerates 249 ISO 3166-1 alpha-2 codes. AIP-143 asks for a
string, and that is reason enough: the list belongs to ISO, changes without
reference to this schema, and a country missing from a generated enum is a
country the API cannot express.

The enum was also unrepresentable downstream, which is how the issue was
found. Cap'n Proto scopes enumerants inside their enum and strips the shared
prefix, so `COUNTRY_CODE_AS` becomes `as` and `COUNTRY_CODE_IN` becomes `in`
-- both keywords -- and the generator's escape, `as_`, is rejected in turn
because Cap'n Proto forbids underscores in declaration names. Neither form
compiled.

## Floats widen to `double`

GraphQL's `Float` is an IEEE 754 double and the source model says nothing
narrower. Halving the wire size by emitting `float` would discard precision
the specification does not say is absent, so the lossless choice was taken.

## No in-progress charging session

`ChargingSession` requires both `start_time` and `end_time`, following the
source model, which types both non-null. A session still running is therefore
not representable, and that is deliberate rather than an oversight: adding an
optional `end_time` would make every reader handle a state the specification
does not define, and the obvious encoding of "in progress" -- an absent end
time -- is indistinguishable from a truncated record.

If COVESA adds a lifecycle state, this schema should follow it rather than
invent one first. Rule 13 of protobuf-rfc, do not pre-build.

## `ChargingPointLabel` keeps its three values

The source model enumerates `POINT_A`, `POINT_B` and `POINT_C`. That is
plainly a placeholder for however a real operator labels its points, and it
was left alone anyway: guessing at the real vocabulary would put values in
the schema that no producer emits. Extend it when a consumer needs it.

## One custom annotation vocabulary, and no more

protobuf-rfc bans custom options outright, and the ban is right for a schema
whose provenance is a document reference. It is wrong here for exactly one
fact: a signal's unit. `speed` is a `double`; nothing in protobuf's type
system says kilometres per hour, and a consumer reading the number without
the unit has read a different quantity.

The vocabulary is therefore limited to what protobuf cannot express -- unit,
quantity kind, VSS element type, fully qualified name -- and every one of
those is *also* written into the doc comment, because an option is invisible
in generated documentation. Nothing else may be added on this precedent.

## The annotation vocabulary is not in the serialization layers

`SignalOptions` and `BranchOptions` describe other schemas. They travel in
the descriptor as options, never on the wire as data, so a FlatBuffers table
for them would be as meaningless as one for `buf.validate.StringRules`. Both
are dropped by `scripts/schema.sh`.

`google/` is kept, because `google.protobuf.Timestamp` and the value types
are real data that fields embed.

## The generator refuses rather than guesses

`sync` parses only the SDL subset `vdm/spec` uses and errors on
anything else; an unknown type is an error, not a skipped field; a source
field colliding with a resource's own AIP fields is an error naming the field,
not a silent overwrite.

The input is a generated, regular corpus. A parser that quietly accepts an
unfamiliar shape produces a schema that is wrong in a way no linter catches,
which is the failure mode this whole repository is arranged to prevent.
