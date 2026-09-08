#!/usr/bin/env sh
# Render the serialization layers: FlatBuffers and Cap'n Proto.
#
# Driven by the `buffers` CLI over a descriptor set rather than by
# protoc-gen-buffers under `buf generate`. The plugin path emits its shared
# `buffers/wellknown` file once per proto file, and buf keeps only the first
# copy -- which holds Timestamp but not Duration, so every schema using a
# Duration then fails to compile. One CLI process writes the file once and
# complete.
#
# The cost of the descriptor set is that imports are rendered too, so the
# dependency trees are removed afterwards: a FlatBuffers table for
# buf.validate.StringRules is meaningless, those are annotation vocabularies.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

buf build -o descriptors.binpb --as-file-descriptor-set

rm -rf schema
out=$(buffers generate --config buffers.yaml 2>&1) || {
	printf '%s\n' "$out" >&2
	exit 1
}
printf '%s\n' "$out"

# The ordinal gate. buffers.yaml explains why it lives here rather than in
# `strict`: the run sees imported schemas this repository does not own, and a
# gap in one of those is not something it can reserve. A diagnostic naming a
# symbol in our own package tree is a real slot shift and fails the build.
if printf '%s\n' "$out" | grep -E '^(warning|error): ordinal:' | grep -q 'protobuf\.covesa\.'; then
	echo "error: an ordinal moved in this repository's own schema:" >&2
	printf '%s\n' "$out" | grep -E '^(warning|error): ordinal:' | grep 'protobuf\.covesa\.' >&2
	exit 1
fi

# protovalidate is an *annotation* vocabulary: its messages state constraints
# on other schemas and carry no data of their own, so a FlatBuffers table for
# buf.validate.StringRules would be meaningless. Dropped.
#
# google/ is kept, and the difference is not arbitrary. google.type.Date and
# google.type.PostalAddress are ordinary value types that fields here embed --
# a vehicle's production date is a Date -- so the emitted schema references
# them and does not compile without them.
rm -rf schema/flatbuffers/buf schema/capnp/buf

# The VSS annotation vocabulary is dropped for the same reason. SignalOptions
# and BranchOptions describe *other* schemas -- which unit a field is in, which
# VSS node a message maps to -- and are carried in the descriptor as options,
# never on the wire as data. A FlatBuffers table for them would be as
# meaningless as one for buf.validate.StringRules.
rm -rf schema/flatbuffers/protobuf/covesa/vss/annotations
rm -rf schema/capnp/protobuf/covesa/vss/annotations

echo "schema: $(find schema/flatbuffers -name '*.fbs' | wc -l | tr -d ' ') .fbs, $(find schema/capnp -name '*.capnp' | wc -l | tr -d ' ') .capnp"
