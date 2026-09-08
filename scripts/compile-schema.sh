#!/usr/bin/env sh
# Compile every emitted schema with the real toolchains.
#
# Separate from scripts/schema.sh because this tests *their* acceptance of the
# output rather than this repository's determinism, and because it needs flatc
# and capnp installed where emitting the schema needs neither.
#
# It is not redundant with `buf build`. A .fbs or .capnp that this repository
# emits can be structurally fine as protobuf and still not compile: an include
# that resolves to nothing, a declaration name a target reserves. Both have
# happened here, and neither is visible to any protobuf tool.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

fail=0
total=0

if [ -d schema/flatbuffers ]; then
	cd schema/flatbuffers
	for f in $(find . -name '*.fbs'); do
		total=$((total + 1))
		if ! flatc --cpp -I . -o "$tmp" "$f" >"$tmp/err" 2>&1; then
			fail=$((fail + 1))
			echo "flatc: $f" >&2
			sed 's/^/    /' "$tmp/err" >&2
		fi
	done
	cd "$root"
fi

if [ -d schema/capnp ]; then
	cd schema/capnp
	for f in $(find . -name '*.capnp'); do
		total=$((total + 1))
		if ! capnp compile -I. -o- "$f" >/dev/null 2>"$tmp/err"; then
			fail=$((fail + 1))
			echo "capnp: $f" >&2
			sed 's/^/    /' "$tmp/err" >&2
		fi
	done
	cd "$root"
fi

if [ "$fail" -ne 0 ]; then
	echo "$fail of $total schemas failed to compile." >&2
	exit 1
fi
echo "compiled: $total schemas, all clean"
