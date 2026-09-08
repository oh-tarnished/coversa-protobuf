#!/usr/bin/env sh
# Build the two proto trees the buffers CLI renders from.
#
# buffers needs every import on disk, and it renders whatever is in its
# `paths` root. So the schema this repository owns and the vocabularies it
# merely imports go into separate roots:
#
#   build/own   -- this repository's protos. Rendered into .fbs and .capnp.
#   build/deps  -- googleapis, protovalidate and the well-known types.
#                  Resolved, never rendered: they are annotation vocabularies,
#                  and a FlatBuffers table for buf.validate.StringRules would
#                  be meaningless.
#
# `buf export` resolves the BSR modules but omits the well-known types, which
# it treats as always-available; protoc ships them, so they are copied from
# there. Without descriptor.proto, protovalidate does not compile.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

rm -rf build
mkdir -p build/own build/deps

buf export . -o build/all
cp -R build/all/protobuf build/own/protobuf
for d in google buf; do
	[ -d "build/all/$d" ] && cp -R "build/all/$d" "build/deps/$d"
done
rm -rf build/all

# The well-known types, from whichever protoc is installed.
wkt=""
for c in /opt/homebrew/include /usr/local/include /usr/include; do
	if [ -f "$c/google/protobuf/descriptor.proto" ]; then
		wkt="$c"
		break
	fi
done
if [ -z "$wkt" ]; then
	echo "error: google/protobuf/descriptor.proto not found; install protobuf" >&2
	exit 1
fi
mkdir -p build/deps/google/protobuf
cp "$wkt"/google/protobuf/*.proto build/deps/google/protobuf/

echo "exported: $(find build/own -name '*.proto' | wc -l | tr -d ' ') own, $(find build/deps -name '*.proto' | wc -l | tr -d ' ') imported"
