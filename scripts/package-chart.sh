#!/usr/bin/env bash
set -euo pipefail

mkdir -p "$usage_output_dir"
source_dir=$(mktemp -d)
trap 'rm -rf "$source_dir"' EXIT
cp -R deploy/charts/handlat/. "$source_dir/"

yq -i '
  .image.repository = strenv(usage_image_repository) |
  .image.digest = strenv(usage_image_digest) |
  .image.tag = ""
' "$source_dir/values.yaml"

helm lint --strict "$source_dir" --namespace handlat \
  --set webAuthn.rpID=preview.example.com --set webAuthn.origin=https://preview.example.com
helm template handlat "$source_dir" --namespace handlat \
  --set webAuthn.rpID=preview.example.com --set webAuthn.origin=https://preview.example.com > /dev/null
helm package "$source_dir" --destination "$usage_output_dir" \
  --version "$usage_version" --app-version "$usage_app_version"
