#!/usr/bin/env bash

set -euo pipefail

# Generates release notes based on the semantic commits between
# the last tag and origin/main, and proposes the next version tag.
#
# Usage:
#   release-notes.sh              # uses current git repo
#   release-notes.sh <git-url>    # clones the repo first

cleanup() {
  if [[ -n "${tmpdir:-}" ]]; then
    rm -rf "$tmpdir"
  fi
}
trap cleanup EXIT

if [[ $# -gt 0 ]]; then
  tmpdir=$(mktemp -d)
  git clone "$1" "$tmpdir" >/dev/null 2>&1
  cd "$tmpdir"
else
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "Error: not in a git repository" >&2
    exit 1
  fi
fi

git fetch origin --tags

latest_tag=$(git -c 'versionsort.suffix=-' tag --list "v*.*.*" --sort=-v:refname | head -n1 2>/dev/null || true)

if [[ -n "$latest_tag" ]]; then
  range="${latest_tag}..origin/main"
  raw_version=${latest_tag#v}
  IFS='.' read -r major minor patch <<< "$raw_version"
  major="${major:-0}"
  minor="${minor:-0}"
  patch="${patch:-0}"
else
  range="origin/main"
  major=0
  minor=0
  patch=0
fi

commits=$(git log --no-merges --format="%s" "$range" | sort 2>/dev/null || true)

if [[ -z "$commits" ]]; then
  echo "No changes since last release."
  exit 0
fi

breaking=()
features=()
fixes=()
other=()

while IFS= read -r msg; do
  [[ -z "$msg" ]] && continue

  semantic_commits_regex='^([a-zA-Z]+)(\([^)]*\))?(!)?: (.*)$'
  if [[ $msg =~ $semantic_commits_regex ]]; then
    type="${BASH_REMATCH[1]}"
    breaking_flag="${BASH_REMATCH[3]}"

    if [[ -n "$breaking_flag" ]]; then
      breaking+=("$msg")
    elif [[ "$type" == "feat" ]]; then
      features+=("$msg")
    elif [[ "$type" == "fix" ]]; then
      fixes+=("$msg")
    else
      other+=("$msg")
    fi
  else
    other+=("$msg")
  fi
done <<< "$commits"

has_breaking=false
has_feat=false
[[ ${#breaking[@]} -gt 0 ]] && has_breaking=true
[[ ${#features[@]} -gt 0 ]] && has_feat=true

if [[ "$major" -eq 0 ]]; then
  if $has_breaking || $has_feat; then
    minor=$((minor + 1))
    patch=0
  else
    patch=$((patch + 1))
  fi
else
  if $has_breaking; then
    major=$((major + 1))
    minor=0
    patch=0
  elif $has_feat; then
    minor=$((minor + 1))
    patch=0
  else
    patch=$((patch + 1))
  fi
fi

print_section() {
  local heading="$1"
  shift
  local items=("$@")
  if [[ ${#items[@]} -gt 0 ]]; then
    echo "### $heading"
    echo
    for item in "${items[@]}"; do
      echo "- $item"
    done
    echo
  fi
}

echo "v${major}.${minor}.${patch}"
echo
print_section "Breaking Changes" "${breaking[@]}"
print_section "Features" "${features[@]}"
print_section "Fixes" "${fixes[@]}"
print_section "Other" "${other[@]}"
