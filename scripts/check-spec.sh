#!/usr/bin/env sh
# Check sync/spec.yaml against the specifications actually checked out.
#
# The pins are what every generated file's banner claims. If a working tree has
# moved and its pin has not, all 270 files carry a revision they were not
# generated from -- a lie that no linter and no compiler can see, and one that
# only matters later, when someone tries to reproduce the output.
#
# Two sources are checked, because the schema has two and they move
# separately: `vss` for the vehicle tree, `vdm` for everything around it.
#
# Each is checked three ways: the commit is the one checked out, the version is
# that commit's date, and the tree is clean so the commit describes it fully.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

# field prints one key from one source block of the pin.
#
# Read with awk rather than a YAML parser so this script depends on nothing:
# it is the first thing CI runs, before any toolchain is known to work.
field() {
	awk -v src="$1:" -v key="$2:" '
		$1 == src { in_block = 1; next }
		/^[^[:space:]#]/ { in_block = 0 }
		in_block && $1 == key { print $2; exit }
	' sync/spec.yaml
}

fail=0

# check verifies one pinned source against its submodule.
check() {
	name=$1
	pin_commit=$(field "$name" commit)
	pin_version=$(field "$name" version)
	pin_path=$(field "$name" path)

	if [ -z "$pin_commit" ] || [ -z "$pin_version" ] || [ -z "$pin_path" ]; then
		echo "error: sync/spec.yaml has no complete '$name' pin." >&2
		fail=1
		return
	fi

	if [ ! -e "$pin_path" ]; then
		echo "error: $name.path is $pin_path, which does not exist." >&2
		echo "       Run: git submodule update --init" >&2
		fail=1
		return
	fi

	# The checkout is found by walking up from the pinned path rather than by
	# assuming where it sits, so moving the submodules -- as they moved into
	# modules/ -- does not silently disable this check. A submodule's .git is
	# a *file* holding a gitlink, not a directory, so test for either.
	dir=$pin_path
	while [ "$dir" != "." ] && [ ! -e "$dir/.git" ]; do
		dir=$(dirname "$dir")
	done

	if [ "$dir" = "." ]; then
		echo "error: $pin_path is not inside a checkout of its own." >&2
		echo "       Run: git submodule update --init" >&2
		fail=1
		return
	fi

	have_commit=$(git -C "$dir" rev-parse HEAD)
	have_version=$(git -C "$dir" log -1 --format=%cd --date=short)

	if [ "$pin_commit" != "$have_commit" ]; then
		echo "error: sync/spec.yaml pins $name at $pin_commit" >&2
		echo "       but $dir is at        $have_commit" >&2
		fail=1
	fi

	if [ "$pin_version" != "$have_version" ]; then
		echo "error: sync/spec.yaml says $name version $pin_version," >&2
		echo "       but that commit is dated $have_version" >&2
		fail=1
	fi

	if [ -n "$(git -C "$dir" status --porcelain)" ]; then
		echo "error: $dir has uncommitted changes, so the pinned commit" >&2
		echo "       does not describe what was read." >&2
		fail=1
	fi

	echo "  $name $pin_version ($(echo "$pin_commit" | cut -c1-7)) <- $dir"
}

echo "spec pins:"
check vss
check vdm

if [ "$fail" -ne 0 ]; then
	echo >&2
	echo "Update the pin, run 'just sync', and record the change in docs/spec.md." >&2
	exit 1
fi
