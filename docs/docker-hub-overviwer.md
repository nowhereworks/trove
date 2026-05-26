# Trove

Trove is a self-hosted registry for agent-consumable engineering artifacts: `AGENTS.md` files, skills, commands, prompts, context packs, schemas, runbooks, templates, and related project assets.

> [!WARNING]
> Trove is pre-v1.0 software. It is highly unstable and subject to heavy changes, including additions, removals, and breaking changes to functionality.
>
> The API contract is not guaranteed at this stage. Use at your own risk.

- Website: https://nowhereworks.github.io/trove/
- Docs: https://nowhereworks.github.io/trove/docs/
- Deployment docs: https://nowhereworks.github.io/trove/docs/operations/deployment.html
- Configuration docs: https://nowhereworks.github.io/trove/docs/operations/configuration.html
- GitHub releases: https://github.com/nowhereworks/trove/releases

## Image

`nowhereworks/trove` is a multi-architecture image for:

- `linux/amd64`
- `linux/arm64`

The image runs Trove as a single Go binary with the embedded React/Vite web UI.

## Quick Start

Pull the Trove image:

```bash
docker pull nowhereworks/trove:latest
```

PostgreSQL is required for server mode; Trove does not support an in-memory database mode.

## Run With PostgreSQL

Create a Docker network:

```bash
docker network create trove-dev
```

Start PostgreSQL:

```bash
docker run -d \
  --name trove-postgres \
  --network trove-dev \
  -e POSTGRES_USER=trove \
  -e POSTGRES_PASSWORD=trove \
  -e POSTGRES_DB=trove \
  -v trove-postgres-data:/var/lib/postgresql/data \
  postgres:16
```

Start Trove with PostgreSQL-backed storage:

```bash
docker run --rm \
  --name trove \
  --network trove-dev \
  -p 8080:8080 \
  -e TROVE_PUBLIC_URL=http://localhost:8080 \
  -e TROVE_DATABASE_URL='postgres://trove:trove@trove-postgres:5432/trove?sslmode=disable' \
  -e TROVE_DATABASE_MIGRATE_ON_STARTUP=true \
  -e TROVE_AUTH_MODE=dev \
  -e TROVE_AUTH_DEV_MODE_ENABLED=true \
  nowhereworks/trove:latest
```

Open http://localhost:8080.

For local development, the default dev bearer token is:

```text
dev-token-local-only
```

## Production Notes

Use an exact version tag in production instead of `latest`:

```bash
docker pull nowhereworks/trove:0.1.0
```

For production deployments:

- Use PostgreSQL 15+.
- Keep `TROVE_DATABASE_MIGRATE_ON_STARTUP=false`.
- Run migrations as an explicit deployment step before serving traffic.
- Use OIDC auth instead of dev auth.
- Set `TROVE_PUBLIC_URL` to the external HTTPS URL.
- Store secrets in your platform secret manager.

Example production-style container command:

```bash
docker run -d \
  --name trove \
  -p 8080:8080 \
  -e TROVE_SERVER_LISTEN=:8080 \
  -e TROVE_PUBLIC_URL=https://trove.example.com \
  -e TROVE_DATABASE_URL='postgres://trove:REPLACE_ME@postgres:5432/trove?sslmode=require' \
  -e TROVE_DATABASE_MIGRATE_ON_STARTUP=false \
  -e TROVE_AUTH_MODE=oidc \
  -e TROVE_AUTH_DEV_MODE_ENABLED=false \
  -e TROVE_OIDC_ISSUER_URL=https://issuer.example.com \
  -e TROVE_OIDC_CLIENT_ID=trove \
  -e TROVE_OIDC_CLIENT_SECRET=REPLACE_ME \
  -e TROVE_OIDC_REDIRECT_URL=https://trove.example.com/auth/oidc/callback \
  nowhereworks/trove:0.1.0
```

## Common Configuration

| Variable | Default | Description |
|---|---:|---|
| `TROVE_SERVER_LISTEN` | `:8080` | Address and port Trove listens on |
| `TROVE_PUBLIC_URL` | unset | Public URL used for redirects and links |
| `TROVE_DATABASE_URL` | unset | PostgreSQL connection URL |
| `TROVE_DATABASE_MIGRATE_ON_STARTUP` | `false` | Run migrations during startup; use only for dev/tests |
| `TROVE_AUTH_MODE` | `dev` | `dev` or `oidc` |
| `TROVE_AUTH_DEV_MODE_ENABLED` | `true` | Enable local development auth mode |
| `TROVE_AUTH_DEV_TOKEN` | `dev-token-local-only` | Static dev bearer token |
| `TROVE_OIDC_ISSUER_URL` | unset | OIDC issuer URL |
| `TROVE_OIDC_CLIENT_ID` | unset | OIDC client ID |
| `TROVE_OIDC_CLIENT_SECRET` | unset | OIDC client secret |
| `TROVE_OIDC_REDIRECT_URL` | unset | OIDC callback URL |
| `TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT` | `true` | Require auth for raw artifact URLs by default |
| `TROVE_REVIEWS_REQUIRE_APPROVAL` | `true` | Require approval before publishing |
| `TROVE_REVIEWS_MINIMUM_APPROVALS` | `1` | Minimum approvals required |
| `TROVE_SECURITY_SECRET_SCANNING` | `true` | Enable secret scanning |
| `TROVE_SECURITY_UNSAFE_INSTRUCTION_SCANNING` | `true` | Enable unsafe-instruction scanning |

Full configuration reference: https://nowhereworks.github.io/trove/docs/operations/configuration.html

## Health Checks

Trove exposes:

```text
/healthz
/readyz
```

Example:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## Use The CLI From The Image

The same image can run Trove CLI commands:

```bash
docker run --rm \
  -e TROVE_SERVER_URL=http://host.docker.internal:8080 \
  nowhereworks/trove:latest \
  help
```

Resolve a package selector:

```bash
docker run --rm \
  -e TROVE_SERVER_URL=http://host.docker.internal:8080 \
  -e TROVE_TOKEN='dev-token-local-only' \
  nowhereworks/trove:latest \
  resolve nwks/platform/agent-backend@stable --json
```

CLI docs: https://nowhereworks.github.io/trove/docs/cli/resolve.html

## Tags

| Source | Tags |
|---|---|
| `main` branch | `latest` |
| `vMAJOR.MINOR.PATCH` release tag | `latest`, `MAJOR.MINOR.PATCH`, `MAJOR.MINOR`, `MAJOR` |

Use `latest` for evaluation and development. Use an exact version tag for production.

## What Trove Provides

- Immutable published package versions using strict SemVer.
- Package references like `org/namespace/package@selector`.
- Server-side selectors for `latest`, `stable`, exact versions, and digests.
- Public, internal, and private visibility rules.
- OIDC human auth, scoped bearer tokens for agents/CI, and dev auth for local use.
- Upload, review, approval, publish, deprecate, and yank workflows.
- Raw artifact, manifest, archive, resolve, search, update check, and compatibility APIs.
- Embedded web UI for discovery, package detail, upload/publish, reviews, and adoption reporting.

Learn more in the Trove docs: https://nowhereworks.github.io/trove/docs/
