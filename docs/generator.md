# The generator

`sync/` is one Go module and one program, in two commands: `sync` writes the
protobuf, `docs` writes the Markdown reference beside it. Both read the same
specification and build the same model.

It has **no dependencies**, and the `require` block should stay empty. The
generator reads GraphQL SDL, a four-key YAML pin and writes text, none of
which needs a library -- and a generator with nothing to keep current is one
that still builds when someone returns to it in two years.

## Laid out by stage
```
sync/cmd/sync/         the CLI that regenerates protobuf/
sync/cmd/docs/         the CLI that regenerates the Markdown reference
sync/internal/spec/    the revision pin
sync/internal/sdl/     the GraphQL parser
sync/internal/naming/  identifier case and pluralisation
sync/internal/catalog/ the AIP rename tables -- data, not logic
sync/internal/model/   what becomes a package, and what its name is
sync/internal/plan/    what one field becomes
sync/internal/emit/    the protobuf text
sync/internal/docs/    the Markdown reference
```

Each package is one stage, and the dependencies only ever point down that
list: `emit` knows about `plan`, `plan` knows about `model`, and nothing knows
about `emit`. A change to how a field is *named* has one place to go; a change
to how it is *written* has another.

`catalog` is the odd one and deliberately so: it is data, not logic. Every
entry is a place VSS vocabulary and an AIP rule disagree, and every one is
also a row in the catalogue at the foot of [`conventions.md`](conventions.md).
It is what a specification bump most often needs edited, so it is kept where
it can be read without reading the code around it.

## The 200-line cap

**Every Go file is capped at 200 lines**, tighter than the 250 prose gets, and
CI checks it. The cap is a prompt: the generator is decomposed by stage, so a
file over it usually means two stages have merged, and the fix is to find the
seam rather than to compress.

## Why there is a workspace

`go.work` names one member, `./sync`. The other `go.mod` in the tree --
`sandbox/go/go.mod` -- is a *template*, copied over `gen/go` in CI so the
generated protobuf Go can be compiled; it holds no packages of its own, and a
workspace member with nothing to build would be a lie.

That omission has a consequence worth knowing: a `go build` run inside
`gen/go` resolves the root workspace and fails, because `gen/go` is not a
member. CI and the justfile set `GOWORK=off` for that one build.

## Tests

`go test ./sync/...` covers the decisions, not the whitespace: what the SDL
parser accepts and refuses, how identifiers are cased and pluralised, which
branches become resources and what their names are, and -- in `emit` -- the
handful of facts the emitted text has to carry. A signal's unit reaching both
the annotation and the comment, a width bound and a `@range` being intersected
into one rule rather than two, the held field numbers being reserved rather
than skipped.

The strongest check is not in the test suite: CI regenerates the whole tree
and diffs it. A change that alters any of the 277 files shows up there
whether or not a test covers it.
