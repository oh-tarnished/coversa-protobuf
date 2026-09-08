# vdm-protobuf dev tasks — run `just` (or `just --list`) to see recipes.
#
# Common flows:
#   just sync     # regenerate protobuf/ from the pinned spec revision
#   just lint     # what CI checks: format, buf lint, build, api-linter
#   just schema   # FlatBuffers + Cap'n Proto, then compile both
#   just ci       # everything CI runs
#
# Requires: go, buf, api-linter. `schema` additionally needs flatc and capnp;
# `langs` needs network access for the BSR's remote plugins.

# Every language template under buf/. A recipe iterates these rather than
# naming them, so adding buf/<lang>.yaml is the whole job of adding a language.
langs := "go java kotlin typescript python cpp csharp rust php ruby objc dart"

# These two need --include-imports and the comments in their templates say why:
# protoc-gen-es writes a relative import per dependency, and generated Python
# does `from buf.validate import validate_pb2`, a path no PyPI package exposes.
vendored := "typescript python"

# List recipes (default when you run bare `just`).
_default:
    @just --list

# Regenerate protobuf/ from the pinned specification revision.
[doc("Regenerate every .proto from the pinned spec. Nothing under protobuf/ is hand-written.")]
sync:
    go run sync/*.go
    buf format -w

# Check sync/spec.yaml against the specification actually checked out.
[doc("Verify the spec revision pin matches the vdm working tree.")]
spec:
    ./scripts/check-spec.sh

# Report what the generator parsed, and every field name tripping an AIP rule.
[doc("Survey the parsed model and its AIP naming traps.")]
survey:
    @go run sync/*.go -survey

# Format Go sources in place.
fmt:
    gofmt -w sync

# Everything the Lint job checks. Mutates nothing.
[doc("Format check, buf lint, buf build, api-linter, and the hand-written line cap.")]
lint: spec aip
    @test -z "$(gofmt -l sync)" || { echo "unformatted Go (run: just fmt):"; gofmt -l sync; exit 1; }
    buf format --diff --exit-code
    buf lint
    buf build
    @just cap

# Run the Google API linter over every proto. No config file, no disabled rule.
[doc("Run api-linter over protobuf/ (zero findings, no suppressions).")]
aip:
    #!/usr/bin/env sh
    set -eu
    d=$(mktemp -d); trap 'rm -rf "$d"' EXIT
    buf build -o "$d/desc.binpb" --as-file-descriptor-set
    api-linter --descriptor-set-in "$d/desc.binpb" \
        --output-format=summary $(find protobuf -name '*.proto')

# The 250-line cap, on files a person actually writes.
#
# Two exemptions, both because the cap's remedy is to *split* and splitting is
# unavailable rather than merely inconvenient:
#
#   protobuf/  generated, and protobuf cannot continue a message across files
#   README.md  the entry document. Its length is diagrams, and a reader who
#              has to follow a link to see how the thing works has been given
#              a worse README, not a shorter one.
#
# Everything else is capped. See CLAUDE.md rule 2.
[doc("Check the 250-line cap on hand-written files.")]
cap:
    #!/usr/bin/env sh
    over=0
    for f in $(find . -path ./gen -prune -o -path ./build -prune -o -path ./schema -prune \
        -o -path ./vdm -prune -o -path ./.git -prune -o -name 'README.md' -prune \
        -o \( -name '*.md' -o -name '*.go' \) -print); do
        n=$(wc -l <"$f")
        [ "$n" -gt 250 ] && { echo "$f: $n lines, over the 250-line cap"; over=1; }
    done
    [ "$over" -eq 0 ] || { echo "Split the files above; do not compress them."; exit 1; }
    echo "All hand-written files within the 250-line cap."

# Verify the committed tree matches what the generator produces.
[doc("Fail if protobuf/ is stale relative to the pinned spec.")]
verify-sync: sync
    @git diff --exit-code -- protobuf/ \
        || { echo "protobuf/ is stale — run 'just sync' and commit"; exit 1; }

# Emit the FlatBuffers and Cap'n Proto schemas, then compile both.
[doc("Emit .fbs and .capnp, then prove flatc and capnp accept them.")]
schema:
    ./scripts/schema.sh
    ./scripts/compile-schema.sh

# Fail if regenerating would move a field's target slot.
[doc("Check the ordinal ledger against a fresh build.")]
verify-schema:
    buffers verify --config buffers.yaml

# Generate one language: `just lang go`, `just lang python`.
[doc("Generate one language from buf/<name>.yaml.")]
lang name:
    #!/usr/bin/env sh
    set -eu
    case " {{vendored}} " in
        *" {{name}} "*) extra=--include-imports ;;
        *) extra= ;;
    esac
    buf generate --template buf/{{name}}.yaml $extra

# Generate every language under buf/.
[doc("Generate every language template in buf/.")]
langs:
    #!/usr/bin/env sh
    set -eu
    for l in {{langs}}; do
        echo "==> $l"
        just lang "$l"
    done

# Remove every generated artefact. buffers.lock is kept: it is committed.
[doc("Remove gen/, schema/, build/ and the descriptor set.")]
clean:
    rm -rf gen schema build descriptors.binpb

# Everything CI runs.
[doc("Everything CI checks.")]
ci: lint verify-sync schema verify-schema
