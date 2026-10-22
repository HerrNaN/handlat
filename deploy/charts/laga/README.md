# Laga Helm chart

Runs one application replica with SQLite at `/data/laga.sqlite` on a persistent
volume. Upgrades use `Recreate`, so brief downtime is expected. No deployment
automation or cluster infrastructure is included.

## Install

Requires Kubernetes 1.26+, Helm 3 or 4, and a provisioner for persistent volumes
(or an existing claim). Ingress requires an ingress controller and HTTPS for
WebAuthn; this chart does not install a controller or issue certificates.

Create a values file for your environment:

```yaml
image:
  # Only needed for a local source-chart install; published devel charts pin this.
  digest: sha256:<actual-image-digest>

webAuthn:
  rpID: laga.example.com
  origin: https://laga.example.com

persistence:
  storageClass: longhorn
  size: 1Gi

ingress:
  enabled: true
  className: traefik
  host: laga.example.com
  tlsSecretName: laga-tls
```

```sh
helm upgrade --install laga ./deploy/charts/laga \
  --namespace laga --create-namespace --values laga-values.yaml --wait
```

Image repository defaults to `ghcr.io/herrnan/laga`. Set `image.tag` to a commit
SHA if not using a digest; a digest takes precedence. The source chart has no
default image version, and `latest` is rejected. For private images, configure
`imagePullSecrets: [{name: ghcr}]` using an existing secret in the release namespace.

Set the RP ID to the public hostname (without scheme or port) and the origin to
the exact public URL (including scheme and any non-standard port). Keep the
ingress host consistent with these settings. TLS terminates at the ingress.

All settings are listed in [values.yaml](values.yaml). Resources have no default
requests/limits; set `resources` to suit your cluster. The container runs as UID/GID
10001 with a read-only root filesystem; the volume must support `fsGroup` or
already be writable by that identity.

## Devel channel

Every push to `main` triggers verification, tests, and publishing. Superseded
in-progress runs are cancelled. Successful runs publish:

- Image: `ghcr.io/herrnan/laga:<commit-sha>` and the moving `devel` tag
  (replacing the previous `latest` tag).
- Chart: `oci://ghcr.io/herrnan/charts/devel/laga`, versioned as
  `<chart-version>-<UTC timestamp YYYYMMDDHHMMSS>` (for example
  `0.1.0-20261018163000`). Each package pins the published image digest in its default
  values and records the commit SHA as `appVersion`.

The workflow summary lists the exact chart version and image digest. Source
`Chart.yaml` and `values.yaml` are not modified by publishing.

The packaging task can also run locally without GitHub environment variables:

```sh
mise package-chart \
  --version "0.1.0-$(date -u +%Y%m%d%H%M%S)" --app-version "$(git rev-parse HEAD)" \
  --image-repository ghcr.io/herrnan/laga --image-digest "sha256:<actual-image-digest>"
```

Packages are written to `_data/charts` by default (override with `--output-dir`).
After authenticating to your registry with `helm registry login`, publish with:

```sh
mise publish-chart _data/charts/laga-0.1.0-20261018163000.tgz \
  oci://ghcr.io/herrnan/charts/devel
```

To install or update to the newest development chart, create environment values
like the example above **without the `image` section**, then run:

```sh
helm upgrade --install laga-devel oci://ghcr.io/herrnan/charts/devel/laga \
  --devel --namespace laga-devel --create-namespace \
  --values laga-devel-values.yaml --wait
```

The registry path selects the channel. Helm's `--devel` flag only allows
prerelease versions, including these timestamp-suffixed versions.

For a reproducible install, replace `--devel` with
`--version 0.1.0-20261018163000`, using a version from a successful run. Do not use
`--reuse-values` when following this channel: it can preserve the old image
digest instead of the new chart default. Updating the `devel` image tag does not
restart existing pods; upgrade the chart to deploy a new build.

GHCR packages may initially be private. For private charts, authenticate with
`helm registry login ghcr.io` before installation. Image pulls need a separate
Kubernetes pull secret via `imagePullSecrets`; Helm's login does not configure
the cluster. OCI publishing uses the workflow's existing `packages: write`
permission. No stable artifacts are published, and no cluster deployment is
triggered by this workflow.

## Storage and upgrades

- Keep exactly one replica; do not attach an HPA or share a claim between releases.
- The database directory, including SQLite's WAL and SHM files, is mounted as a
  whole. Use block-backed storage with SQLite-compatible locking, not an RWX
  network filesystem.
- `ReadWriteOnce` is the default. It limits access to one node, **not one pod**.
  `Recreate` avoids overlap during upgrades, but manual pod deletion can briefly
  overlap pods. Use `persistence.accessMode: ReadWriteOncePod` when your CSI driver
  supports it. Do not force-delete a running pod.
- Empty `persistence.storageClass` uses the cluster default; `"-"` selects no
  storage class. Storage class and access mode are generally immutable after
  claim creation; size increases require storage-driver support.
- Helm retains the PVC on uninstall. It does not protect against deleting the
  namespace, manually deleting the PVC, or losing the underlying storage. Arrange
  consistent database/volume backups separately; do not copy just a live SQLite
  file while ignoring its WAL.
- After uninstall, reinstall with `persistence.existingClaim` set to the retained
  claim's name (normally the release name if it contains `laga`, otherwise
  `<release>-laga`). Helm will not create or manage that claim. Do not change claim
  names on a running installation without an explicit data migration.
- Separate labs and production releases should use separate namespaces and PVCs.
  Helm rollback rolls back manifests, not database contents or schema migrations.

Startup/readiness/liveness probes request `/`; these check HTTP serving, not
database health. Kubernetes SIGTERM triggers the application's existing graceful
HTTP shutdown, with 30 seconds allowed before forced termination.

## Validate without deploying

From the repository root:

```sh
helm lint --strict ./deploy/charts/laga --values ./deploy/charts/laga/ci/values.yaml
helm template laga ./deploy/charts/laga --values ./deploy/charts/laga/ci/values.yaml
```

The CI values contain a placeholder digest for rendering only, not installation.
