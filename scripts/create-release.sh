#!/usr/bin/env bash
set -euo pipefail

# Create GitHub Release based of changes on origin/main since last release/tag

scripts=$(dirname "$(realpath "$0")")

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
tmpfile="$tmpdir/release"
"$scripts/release-notes.sh" > "$tmpfile"
version=$(head -1 "$tmpfile")

if [[ "$version" == "No changes since last release." ]]; then
  echo "$version"
  exit 0
fi

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid release version: $version" >&2
  exit 1
fi

notesfile="$tmpdir/notes"
tail -n+3 "$tmpfile" > "$notesfile"

gh release create "$version" --target "$(git rev-parse origin/main)" \
  --title "$version" --notes-file "$notesfile"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "tag=$version" >> "$GITHUB_OUTPUT"
fi
