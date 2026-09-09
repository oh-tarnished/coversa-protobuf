#!/usr/bin/env sh
# Check that every URL in docs/references.md resolves.
#
# That file exists because CLAUDE.md points at it instead of working from
# memory -- AIP rule semantics have been guessed wrong here more than once.
# A link index nobody checks decays into the thing it was meant to replace,
# so the file says CI verifies it and this is what does.
#
# Only references.md is checked. Every other document links to prose that may
# legitimately move; this one is the index, and a dead entry in it is a
# lookup someone does by hand.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

doc=docs/references.md
fail=0
checked=0

# Bare URLs, one per line, deduplicated. The file writes them unbracketed --
# `AIP-131 <https://aip.dev/131>` in prose, a bare URL in the tables -- so a
# grep for the scheme finds them all without a Markdown parser.
urls=$(grep -oE 'https?://[^ )>|]+' "$doc" | sed 's/[.,]$//' | sort -u)

for url in $urls; do
	checked=$((checked + 1))

	# --location because aip.dev redirects, and a HEAD first because most of
	# these are large HTML pages. Some hosts refuse HEAD, so a failure falls
	# back to a ranged GET before it is believed.
	code=$(curl -sS -o /dev/null -w '%{http_code}' -I -L --max-time 25 "$url" 2>/dev/null || echo 000)
	if [ "$code" -lt 200 ] || [ "$code" -ge 400 ]; then
		code=$(curl -sS -o /dev/null -w '%{http_code}' -r 0-0 -L --max-time 25 "$url" 2>/dev/null || echo 000)
	fi

	if [ "$code" -lt 200 ] || [ "$code" -ge 400 ]; then
		echo "  $code  $url" >&2
		fail=$((fail + 1))
	fi
done

if [ "$fail" -ne 0 ]; then
	echo >&2
	echo "error: $fail of $checked links in $doc do not resolve." >&2
	echo "       Fix the link or drop the entry; do not leave it dangling." >&2
	exit 1
fi

echo "$doc: $checked links, all resolve"
