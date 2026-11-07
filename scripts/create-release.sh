#!/usr/bin/env bash

# Create GitHub Release based of changes on origin/main since last release/tag

scripts=$(dirname "$(realpath "$0")")

tmpfile=$(mktemp)
"$scripts/release-notes.sh" > "$tmpfile"
version=$(head -1 "$tmpfile")

notesfile=$(mktemp)
tail -n+3 "$tmpfile" > "$notesfile"

gh release create "$version" --title "$version" --notes-file "$notesfile"
