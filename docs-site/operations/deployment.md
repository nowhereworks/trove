# Deployment

## Why

Trove is designed to be simple to deploy: a single Go binary and a PostgreSQL database. No external dependencies, no complex orchestration. This page covers how to run Trove in development and production.

## How

### Architecture

```
┌─────────────────────────┐
│     Go Single Binary     │
│  ┌────────────────────┐  │
│  │   JSON APIs         │  │
│  │   Raw Endpoints     │  │
│  │   Embedded SPA UI   │  │
│  │   Health / Metrics  │  │
│  │   CLI               │  │
│  └────────────────────┘  │
└──────────┬───────────────┘
           │
┌──────────▼───────────────┐
│      PostgreSQL           │
│  Metadata + Blob Storage  │
└──────────────────────────┘
```

### Development Setup

#### Prerequisites

- Go 1.26+
- PostgreSQL 15+
- Node.js 20+ (for building the UI)

#### Quick Start

```bash
# 1. Start PostgreSQL
docker run -d --name trove-postgres \
  -e POSTGRES_USER=trove \
  -e POSTGRES_PASSWORD=trove \
  -e POSTGRES_DB=trove \
  -p 5432:5432 \
  postgres:16

# 2. Run migrations
export TROVE_DATABASE_URL="postgres://trove:trove@localhost:5432/trove?sslmode=disable"

# 3. Build the UI
cd web && npm install && npm run build && cd ..

# 4. Run the server (with dev auth mode)
go run cmd/trove/main.go --config config.dev.yaml
```

#### Dev Configuration

```yaml
# config.dev.yaml
server:
  listen: ":8080"
  publicUrl: "http://localhost:8080"

database:
  url: "postgres://trove:trove@localhost:5432/trove?sslmode=disable"
  migrateOnStartup: true

auth:
  mode: dev
  devModeEnabled: true

raw:
  requireAuthByDefault: false

reviews:
  requireApproval: false
```

### Production Deployment

#### Build the Binary

```bash
# Build the UI first
cd web && npm install && npm run build && cd ..

# Build the Go binary
go build -o trove ./cmd/trove
```

#### Published CLI Builds

Trove publishes CLI binaries to GitHub Releases for rolling and tagged releases.

The release workflow builds `trove` for Linux, macOS, and Windows on `amd64` and `arm64`, then uploads the archives plus `checksums.txt`.

| Source | GitHub Release |
|---|---|
| `main` branch | Rolling prerelease tagged `latest`; replaced when CLI-related files change |
| `vMAJOR.MINOR.PATCH` tag | Immutable release tagged `vMAJOR.MINOR.PATCH` |

#### Container Image

Trove publishes a multi-architecture Docker Hub image for `linux/amd64` and `linux/arm64`:

```bash
docker pull nowhereworks/trove:latest
```

Run the image with PostgreSQL-backed storage:

```bash
docker run --rm -p 8080:8080 \
  -e TROVE_SERVER_LISTEN=:8080 \
  -e TROVE_DATABASE_URL="postgres://trove:trove@host.docker.internal:5432/trove?sslmode=disable" \
  -e TROVE_DATABASE_MIGRATE_ON_STARTUP=false \
  nowhereworks/trove:latest
```

Use an exact version tag in production rather than `latest`:

```bash
docker pull nowhereworks/trove:0.1.0
```

The image release workflow publishes these tags:

| Source | Tags |
|---|---|
| `main` branch | `latest` |
| `vMAJOR.MINOR.PATCH` tag | `latest`, `MAJOR.MINOR.PATCH`, `MAJOR.MINOR`, `MAJOR` |

#### Deploy With Helm

The Helm chart deploys only the Trove application. PostgreSQL is external and must already exist.

Use the GHCR-published OCI chart for normal installs:

```bash
helm install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.1 \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

Use the local chart while developing chart changes:

```bash
helm install trove ./charts/trove \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

Use `database.migrateOnStartup=true` only for quick development-style installs. Production should run migrations explicitly before starting or rolling Trove pods.

Prefer an existing Kubernetes Secret for database credentials:

```bash
kubectl create secret generic trove-database \
  --from-literal=TROVE_DATABASE_URL='postgres://trove:trove@postgres.example:5432/trove?sslmode=require'

helm install trove ./charts/trove \
  --set database.existingSecret.name=trove-database
```

Enable Ingress or Gateway API routing through `values.yaml`. Example Ingress install:

```bash
helm install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.1 \
  --set database.existingSecret.name=trove-database \
  --set config.publicUrl=https://trove.nwks.com \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=trove.nwks.com
```

Before exposing Trove outside the cluster, configure OIDC with `auth.mode=oidc` and the `oidc.*` values, or set a non-default dev token for non-production testing.

See [Helm Chart](/operations/helm-chart) for GHCR usage, upgrade commands, production values files, Gateway API examples, chart versioning, and the full `values.yaml` reference.

#### Run Migrations

```bash
# Run migrations explicitly (not on startup)
trove migrate --database-url "$DATABASE_URL"
```

#### Start the Server

```bash
./trove --config /etc/trove/config.yaml
```

#### Production Configuration

```yaml
server:
  listen: ":8080"
  publicUrl: "https://trove.nwks.com"

database:
  url: "${DATABASE_URL}"
  migrateOnStartup: false

auth:
  mode: oidc
  oidc:
    issuerUrl: "${OIDC_ISSUER_URL}"
    clientId: "${OIDC_CLIENT_ID}"
    clientSecretRef: "TROVE_OIDC_CLIENT_SECRET"
    redirectUrl: "https://trove.nwks.com/auth/oidc/callback"

storage:
  mode: postgres
  limits:
    maxArtifactFileBytes: 10485760
    maxUnpackedPackageBytes: 104857600
    maxArtifactsPerVersion: 1000

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

### Health Endpoints

| Path | Purpose | Response |
|---|---|---|
| `/healthz` | Process liveness | `{"status": "ok"}` |
| `/readyz` | Database-backed readiness | `{"status": "ok"}` or 503 |

Use these for Kubernetes health probes or load balancer checks:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

### Metrics

The `/metrics` endpoint exposes Prometheus-compatible metrics:

- API request count, duration, and status
- Raw artifact request count, duration, and status
- Publish attempts and failures
- Validation failures by reason
- Auth failures
- Update check count

### Migration Strategy

| Environment | Migration Approach |
|---|---|
| Development | `migrateOnStartup: true` — runs on server start |
| Production | Explicit `trove migrate` command before starting servers |
| CI/Tests | `migrateOnStartup: true` in test containers |

Down migrations are provided for local development when safe. Production rollbacks are an operational decision.

### Next Steps

- See [Configuration](/operations/configuration) for all config options
- Learn about [Compatibility](/operations/compatibility) rules
