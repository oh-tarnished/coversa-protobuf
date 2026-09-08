#!/usr/bin/env sh
# Check sync/spec.yaml against the specification actually checked out.
#
# The pin is what every generated file's banner claims. If the vdm working
# tree has moved and the pin has not, all 276 files carry a revision they were
# not generated from -- a lie that no linter and no compiler can see, and one
# that only matters later, when someone tries to reproduce the output.
#
# Checks three things: the commit is the one checked out, the version is that
# commit's date, and the tree is clean so the commit describes it fully.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

pin_commit=$(sed -n 's/^commit:[[:space:]]*//p' sync/spec.yaml)
pin_version=$(sed -n 's/^version:[[:space:]]*//p' sync/spec.yaml)
pin_path=$(sed -n 's/^path:[[:space:]]*//p' sync/spec.yaml)

spec_root=${pin_path%/spec}
if [ ! -d "$spec_root/.git" ]; then
	echo "note: $spec_root is not a git checkout; cannot verify the pin." >&2
	echo "      Skipping. CI checks this where the checkout is complete."
	exit 0
fi

have_commit=$(git -C "$spec_root" rev-parse HEAD)
have_version=$(git -C "$spec_root" log -1 --format=%cd --date=short)

fail=0

if [ "$pin_commit" != "$have_commit" ]; then
	echo "error: sync/spec.yaml pins   $pin_commit" >&2
	echo "       but $spec_root is at $have_commit" >&2
	fail=1
fi

if [ "$pin_version" != "$have_version" ]; then
	echo "error: sync/spec.yaml says version $pin_version," >&2
	echo "       but that commit is dated $have_version" >&2
	fail=1
fi

if [ -n "$(git -C "$spec_root" status --porcelain)" ]; then
	echo "error: $spec_root has uncommitted changes, so the pinned commit" >&2
	echo "       does not describe what was read." >&2
	fail=1
fi

if [ "$fail" -ne 0 ]; then
	echo >&2
	echo "Update the pin, run 'just sync', and record the change in docs/spec.md." >&2
	exit 1
fi

echo "spec pin is current: $pin_version ($(echo "$pin_commit" | cut -c1-7))"
