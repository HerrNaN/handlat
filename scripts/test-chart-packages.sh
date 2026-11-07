#!/usr/bin/env bash
set -euo pipefail

test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT
export usage_version=0.1.0-pr.42.1.1
export usage_app_version=0123456789abcdef0123456789abcdef01234567
export usage_image_repository=ghcr.io/herrnan/handlat
export usage_image_digest=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export usage_output_dir="$test_dir"

bash scripts/package-chart.sh
chart="$test_dir/handlat-$usage_version.tgz"
values=(--set webAuthn.rpID=preview.example.com --set webAuthn.origin=https://preview.example.com)

helm template handlat "$chart" --namespace handlat-labs "${values[@]}" > "$test_dir/default.yaml"
[[ $(yq 'select(.kind == "PersistentVolumeClaim") | .metadata.annotations."helm.sh/resource-policy"' "$test_dir/default.yaml") == keep ]]
[[ $(helm show values "$chart" | yq '.persistence.retain') == true ]]
[[ $(helm show values "$chart" | yq '.persistence.storageClass') == '' ]]
[[ $(yq 'select(.kind == "Deployment") | .spec.template.spec.containers[0].image' "$test_dir/default.yaml") == "$usage_image_repository@$usage_image_digest" ]]

helm template handlat "$chart" "${values[@]}" --set persistence.retain=true > "$test_dir/retained.yaml"
[[ $(yq 'select(.kind == "PersistentVolumeClaim") | .metadata.annotations."helm.sh/resource-policy"' "$test_dir/retained.yaml") == keep ]]

helm template handlat-pr-42 "$chart" --namespace handlat-pr-42 "${values[@]}" \
  --set persistence.retain=false --set persistence.storageClass=handlat-preview-local > "$test_dir/preview.yaml"
[[ $(yq 'select(.kind == "PersistentVolumeClaim") | .metadata.annotations."helm.sh/resource-policy"' "$test_dir/preview.yaml") == null ]]
[[ $(yq 'select(.kind == "PersistentVolumeClaim") | .spec.storageClassName' "$test_dir/preview.yaml") == handlat-preview-local ]]

helm template handlat "$chart" "${values[@]}" \
  --set persistence.retain=false --set persistence.existingClaim=existing-data > "$test_dir/existing.yaml"
[[ -z $(yq 'select(.kind == "PersistentVolumeClaim")' "$test_dir/existing.yaml") ]]
[[ $(yq 'select(.kind == "Deployment") | .spec.template.spec.volumes[0].persistentVolumeClaim.claimName' "$test_dir/existing.yaml") == existing-data ]]

echo "Chart storage retention checks passed"
