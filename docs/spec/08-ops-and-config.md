# Operations And Configuration

## Deployment Shape

MVP deployment:

```text
Single Go binary
PostgreSQL
Optional object storage later
```

The Go binary serves:

- JSON APIs
- raw artifact endpoints
- embedded SPA UI
- health checks
- metrics

## Backend Stack

Preferred backend choices:

- Go
- `chi`
- `pgx` for PostgreSQL
- `sqlc` for SQL-first DB access
- `go:embed` for UI assets
- OIDC middleware or `go-chi/jwtauth` depending on router choice

Avoid `go-git` in MVP unless Git-backed storage is explicitly added.

## Example Config

```yaml
server:
  listen: ":8080"
  publicUrl: "https://agenthub.company.com"

database:
  url: "postgres://agenthub:agenthub@postgres:5432/agenthub?sslmode=disable"
  migrateOnStartup: false

auth:
  mode: oidc
  devModeEnabled: false
  oidc:
    issuerUrl: "https://login.microsoftonline.com/.../v2.0"
    clientId: "agenthub"
    clientSecretRef: "AGENTHUB_OIDC_CLIENT_SECRET"
    redirectUrl: "https://agenthub.company.com/auth/oidc/callback"
    scopes:
      - openid
      - profile
      - email
  tokens:
    hashSecretRef: "AGENTHUB_TOKEN_HASH_SECRET"

storage:
  mode: postgres
  limits:
    maxArtifactFileBytes: 10485760
    maxUnpackedPackageBytes: 104857600
    maxArtifactsPerVersion: 1000
  s3:
    enabled: false
    endpoint: "http://rustfs:9000"
    bucket: "agenthub-artifacts"
    accessKeyRef: "AGENTHUB_S3_ACCESS_KEY"
    secretKeyRef: "AGENTHUB_S3_SECRET_KEY"

raw:
  requireAuthByDefault: true
  allowPublicNamespaces: true
  allowPublicPackages: true

reviews:
  requireApproval: true
  minimumApprovals: 1
  allowSelfApproval: false

security:
  secretScanning: true
  unsafeInstructionScanning: true
```

`auth.mode: dev` is allowed only for local development and tests. Production deployments must use `auth.mode: oidc`.

OIDC is standards-based and provider-neutral. Provider-specific group/claim mapping can be added later without changing the core login contract.

`storage.mode: postgres` is the MVP default. `storage.s3` reserves the near-future RustFS/S3-compatible path and must not be required for Slice 1.

`database.migrateOnStartup` is for local development and tests only. Production deployments should run migrations as an explicit command or deployment job.

## Health And Metrics

Required endpoints:

| Path | Purpose |
|---|---|
| `/healthz` | process liveness |
| `/readyz` | database-backed readiness |
| `/metrics` | Prometheus/OpenMetrics metrics |

Important metrics:

- API request count, duration, and status
- raw artifact request count, duration, and status
- cache hit/miss if caching is added
- publish attempts and failures
- validation failures by reason
- auth failures
- update check count

## Caching

Exact raw artifact URLs are immutable and can be cached aggressively.

Alias responses are mutable and must use no-cache behavior.

Use ETags based on artifact digest.

Generated exact-version archives use ETags based on the package digest.

## Archive Generation

Archive downloads are generated on demand from artifact rows and blob content.

Supported formats:

- `.tar.gz`
- `.zip`

Archive entry ordering must be deterministic by sorting artifact paths.

## Docker Compose For Development

Expected local services once implementation exists:

- `agenthub`
- `postgres`
- optional `minio`

Do not add Docker Compose as a requirement until the implementation needs it.
