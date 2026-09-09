# References

A checked link index. `scripts/check-links.sh` verifies every URL here
resolves — on change, and weekly, because a link rots without anyone touching
the repository. Run it with `just links`.

The check exists because `CLAUDE.md` points at this file instead of working
from memory, and AIP rule semantics have been guessed wrong here more than
once. A dead entry sends the reader back to guessing.

## The source model

- COVESA Vehicle Data Model — https://github.com/COVESA/vdm
- Simplified Semantic Data Modeling (S2DM) — https://covesa.github.io/s2dm/
- Vehicle Signal Specification — https://covesa.github.io/vehicle_signal_specification/
- VSS rule set (branches, data entry types, instances) — https://covesa.github.io/vehicle_signal_specification/rule_set/
- VSS unit catalogue — https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/units.yaml
- VSS quantity catalogue — https://github.com/COVESA/vehicle_signal_specification/blob/master/spec/quantities.yaml
- GraphQL specification (the SDL the model is written in) — https://spec.graphql.org

Read, but not generated from — see [`decisions.md`](decisions.md):

- Vehicle Information Service Specification (VISS), an access protocol rather than a data model — https://github.com/COVESA/vehicle-information-service-specification
- Commercial Vehicle Information Specifications (CVIS), `.vspec` trees for bus, truck and trailer — https://github.com/COVESA/commercial-vehicle-information-specifications

## The projects this is built on

- protobuf-rfc, whose conventions this repository adapts — https://github.com/the-protobuf-project/protobuf-rfc
- buffers, the FlatBuffers / Cap'n Proto generator — https://github.com/the-protobuf-project/buffers

## AIP

The rules cited in the schema, and the linter page for each.

| AIP | Subject | Linter |
|---|---|---|
| https://aip.dev/121 | Resource-oriented design | https://linter.aip.dev/121/ |
| https://aip.dev/122 | Resource names | https://linter.aip.dev/122/ |
| https://aip.dev/123 | Resource types | https://linter.aip.dev/123/ |
| https://aip.dev/126 | Enumerations | https://linter.aip.dev/126/ |
| https://aip.dev/131 | Get | https://linter.aip.dev/131/ |
| https://aip.dev/132 | List | https://linter.aip.dev/132/ |
| https://aip.dev/133 | Create | https://linter.aip.dev/133/ |
| https://aip.dev/134 | Update | https://linter.aip.dev/134/ |
| https://aip.dev/135 | Delete | https://linter.aip.dev/135/ |
| https://aip.dev/140 | Field names | https://linter.aip.dev/140/ |
| https://aip.dev/141 | Quantities | https://linter.aip.dev/141/ |
| https://aip.dev/142 | Time and duration | https://linter.aip.dev/142/ |
| https://aip.dev/143 | Standardized codes | https://linter.aip.dev/143/ |
| https://aip.dev/148 | Standard fields | https://linter.aip.dev/148/ |
| https://aip.dev/154 | Resource freshness (etag) | https://linter.aip.dev/154/ |
| https://aip.dev/156 | Singleton resources | https://linter.aip.dev/156/ |
| https://aip.dev/158 | Pagination | https://linter.aip.dev/158/ |
| https://aip.dev/160 | Filtering | *(no linter rules)* |
| https://aip.dev/164 | Soft delete and undelete | https://linter.aip.dev/164/ |
| https://aip.dev/191 | File and directory structure | https://linter.aip.dev/191/ |
| https://aip.dev/192 | Documentation | https://linter.aip.dev/192/ |
| https://aip.dev/215 | API-specific protos | https://linter.aip.dev/215/ |
| https://aip.dev/216 | States | https://linter.aip.dev/216/ |

- The linter itself — https://github.com/googleapis/api-linter

The VS Code Google API Linter extension reads `workspace.protobuf.yaml` rather
than the CLI's arguments, so it may report differently from `just aip`. The
workflow is the authority.

## Tooling

- buf — https://buf.build/docs
- buf lint rules — https://buf.build/docs/lint/rules/
- managed mode — https://buf.build/docs/generate/managed-mode/
- protovalidate — https://github.com/bufbuild/protovalidate
- protovalidate rule reference — https://buf.build/bufbuild/protovalidate/docs/main:buf.validate
- FlatBuffers schema grammar — https://flatbuffers.dev/schema/
- Cap'n Proto schema language — https://capnproto.org/language.html
- Cap'n Proto implementations, by language — https://capnproto.org/otherlang.html
- just, the task runner — https://github.com/casey/just

## Protobuf itself

- Language guide (proto3) — https://protobuf.dev/programming-guides/proto3/
- Well-known types — https://protobuf.dev/reference/protobuf/google.protobuf/
- `google.type` — https://github.com/googleapis/googleapis/tree/master/google/type
- Go generated code reference — https://protobuf.dev/reference/go/go-generated/
