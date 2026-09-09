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

# Where capnp's own schema files live.
#
# Every emitted .capnp opens with `using Cxx = import "/capnp/c++.capnp"`, an
# absolute import resolved against the import path rather than the file's
# directory. Homebrew's capnp has its include directory compiled in and finds
# it unaided; Debian and Ubuntu split the compiler (`capnproto`) from the
# schema files (`libcapnp-dev`), so a runner with only the first fails every
# file with "Import failed: /capnp/c++.capnp" -- 267 identical errors that say
# nothing about the schema.
#
# So it is located rather than assumed, and its absence is reported as the
# missing package it is.
capnp_include=""
for dir in \
	"$(dirname "$(command -v capnp 2>/dev/null || echo /nonexistent)")/../include" \
	/usr/include /usr/local/include /opt/homebrew/include; do
	if [ -f "$dir/capnp/c++.capnp" ]; then
		capnp_include=$(CDPATH= cd -- "$dir" && pwd)
		break
	fi
done

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
	if [ -z "$capnp_include" ]; then
		echo "error: capnp/c++.capnp not found on any include path." >&2
		echo "       Install the schema files: apt install libcapnp-dev," >&2
		echo "       or brew install capnp." >&2
		exit 1
	fi

	cd schema/capnp
	for f in $(find . -name '*.capnp'); do
		total=$((total + 1))
		if ! capnp compile -I. -I"$capnp_include" -o- "$f" >/dev/null 2>"$tmp/err"; then
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
