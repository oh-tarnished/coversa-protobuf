# coversa-protobuf dev tasks — run `just` (or `just --list`) to see recipes.
#
# Common flows:
#   just sync     # regenerate protobuf/ from the pinned spec revision
#   just docs     # regenerate the Markdown reference beside the protos
#   just manifest # regenerate the VSS mapping the codec reads
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
    go run ./sync/cmd/sync
    buf format -w
    @just docs
    @just manifest

# Render the Markdown reference: one README per package, plus an index.
[doc("Regenerate the Markdown reference from the pinned spec.")]
docs:
    @go run ./sync/cmd/docs

# Regenerate codec/manifest.json, the language-neutral VSS mapping.
[doc("Regenerate the VSS mapping manifest.")]
manifest:
    @go run ./codec/cmd/manifest

# Fail if the manifest is stale, changing nothing.
[doc("Check the VSS mapping manifest is current.")]
verify-manifest:
    @go run ./codec/cmd/manifest -check

# Check every URL in docs/references.md resolves. Needs network access.
[doc("Check the reference link index.")]
links:
    ./scripts/check-links.sh

# Check sync/spec.yaml against the specification actually checked out.
[doc("Verify the spec revision pins match the checkouts under modules/.")]
spec:
    ./scripts/check-spec.sh

# Report what the generator parsed, and every field name tripping an AIP rule.
[doc("Survey the parsed model and its AIP naming traps.")]
survey:
    @go run ./sync/cmd/sync -survey

# Format Go sources in place.
fmt:
    gofmt -w sync codec

# Build and test the generator.
[doc("Build and vet the generator, and run its tests.")]
test:
    go build ./sync/... ./codec/...
    go vet ./sync/... ./codec/...
    go test ./sync/... ./codec/...

# Everything the Lint job checks. Mutates nothing.
[doc("Format check, buf lint, buf build, api-linter, and the hand-written line cap.")]
lint: spec aip test
    @test -z "$(gofmt -l sync codec)" || { echo "unformatted Go (run: just fmt):"; gofmt -l sync codec; exit 1; }
    buf format --diff --exit-code
    buf lint
    buf build
    @just cap

# Run the Google API linter over every proto. No config file, no disabled rule.
#
# api-linter exits 0 even when it reports problems -- it is a reporter, and
# the CI action is what turns findings into a failure. A local recipe that
# printed a table of violations and still succeeded is worse than no recipe:
# `just ci` passed while CI failed on the same 16 findings. So the summary is
# captured and a non-empty table is made the failure it already was.
[doc("Run api-linter over protobuf/ (zero findings, no suppressions).")]
aip:
    #!/usr/bin/env sh
    set -eu
    d=$(mktemp -d); trap 'rm -rf "$d"' EXIT
    buf build -o "$d/desc.binpb" --as-file-descriptor-set
    api-linter --descriptor-set-in "$d/desc.binpb" \
        --output-format=summary $(find protobuf -name '*.proto') | tee "$d/out"
    # The summary prints a header row whether or not anything was found, so
    # the test is for a data row: a line naming a rule.
    if grep -q '| core::' "$d/out"; then
        echo "api-linter reported violations above. Rule 1: fix the generator," >&2
        echo "never the emitted file, and never except the rule." >&2
        exit 1
    fi

# The 250-line cap, on files a person actually writes.
#
# Two exemptions, both because the cap's remedy is to *split* and splitting is
# unavailable rather than merely inconvenient:
#
#   protobuf/  generated -- both the .proto files, which protobuf cannot
#              continue across files, and the READMEs beside them
#   README.md  the entry document. Its length is diagrams, and a reader who
#              has to follow a link to see how the thing works has been given
#              a worse README, not a shorter one.
#   docs/sample-reference.md
#              a specimen of what the docs generator emits. Exempt for the
#              same reason protobuf/ is: it stands in for generated output,
#              and splitting it would misrepresent what a real page looks
#              like.
#
# Everything else is capped. See CLAUDE.md rule 2.
[doc("Check the line caps: 200 for Go, 250 for Markdown.")]
cap:
    #!/usr/bin/env sh
    over=0
    for f in $(find . -path ./gen -prune -o -path ./build -prune -o -path ./schema -prune \
        -o -path ./modules -prune -o -path ./.git -prune -o -path ./protobuf -prune \
        -o -name 'README.md' -prune -o -name 'sample-reference.md' -prune \
        -o \( -name '*.md' -o -name '*.go' \) -print); do
        n=$(wc -l <"$f")
        # Go is capped tighter than prose: a 200-line file is one a reader can
        # hold in their head, and the generator is decomposed by *stage*, so a
        # file over the cap usually means two stages have merged.
        case "$f" in
            *.go) cap=200 ;;
            *)    cap=250 ;;
        esac
        [ "$n" -gt "$cap" ] && { echo "$f: $n lines, over the $cap-line cap"; over=1; }
    done
    [ "$over" -eq 0 ] || { echo "Split the files above; do not compress them."; exit 1; }
    echo "All hand-written files within their line cap."

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

# Fail if regenerating would move a field's target slot, or if the committed
# .fbs and .capnp are not what the current schema produces.
#
# Two checks, and they catch different things. The ledger catches a slot that
# moved, which is a wire-format break. The diff catches a committed file that
# no longer matches the .proto beside it -- which is what makes shipping the
# schemas safe: a consumer reading schema/ is reading the current schema, not
# whatever was emitted the last time someone remembered to run it.
[doc("Check the ordinal ledger and the committed schema against a fresh build.")]
verify-schema: schema
    buffers verify --config buffers.yaml
    @git diff --exit-code -- schema/ \
        || { echo "schema/ is stale — run 'just schema' and commit"; exit 1; }

# Generate one language: `just lang go`, `just lang python`.
#
# The output directory is removed first. `buf generate` writes over what it
# produces but never deletes what it no longer produces, so a file for a
# message the schema dropped survives every later run -- and then fails the
# build with an import nothing provides, long after the change that caused it.
# CI checks out fresh and never sees this; a working copy does.
[doc("Generate one language from buf/<name>.yaml.")]
lang name:
    #!/usr/bin/env sh
    set -eu
    case " {{vendored}} " in
        *" {{name}} "*) extra=--include-imports ;;
        *) extra= ;;
    esac
    rm -rf gen/{{name}}
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

# Compile the generated Go, proving the schema produces buildable code.
#
# GOWORK=off because gen/go is not a workspace member: go.work names only
# ./sync, and a build inside a non-member module fails while the workspace is
# active. See the comment in go.work.
[doc("Generate Go and compile it, the way CI does.")]
build-go: (lang "go")
    cp sandbox/go/go.mod gen/go/go.mod
    cd gen/go && GOWORK=off GOFLAGS=-mod=mod go mod tidy && GOWORK=off go build ./...

# Remove every generated artefact. buffers.lock is kept: it is committed.
[doc("Remove gen/, schema/, build/ and the descriptor set.")]
clean:
    rm -rf gen schema build descriptors.binpb

# Everything CI runs.
[doc("Everything CI checks.")]
ci: lint verify-sync verify-manifest schema verify-schema
