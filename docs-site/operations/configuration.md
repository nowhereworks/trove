# Configuration

## Why

Trove needs startup configuration for HTTP serving, PostgreSQL, authentication, raw artifact access, review policy, and upload limits. Trove supports loading configuration from environment variables, a YAML config file, or both combined.

## How

### Configuration Precedence

Trove layers configuration in this order:

1. **Defaults** — built-in sensible defaults
2. **YAML config file** — optional, loaded via `trove server --config <path>`
3. **Environment variables** — always take final precedence

You can use env vars only, a config file only, or both. When both are used, env vars override any keys set in the file.

### Config File

Start the server with a YAML config file:

```bash
trove server --config /etc/trove/config.yaml
```

A minimal config only needs the keys you want to override from defaults:

```yaml
database:
  url: "postgres://trove:trove@localhost:5432/trove?sslmode=disable"
  migrateOnStartup: true
```

Only non-empty values from the file are applied. You do not need to specify every key.

### Development Example (Env Vars)

```bash
export TROVE_SERVER_LISTEN=:8080
export TROVE_PUBLIC_URL=http://localhost:8080
export TROVE_DATABASE_URL="postgres://trove:trove@localhost:5432/trove?sslmode=disable"
export TROVE_DATABASE_MIGRATE_ON_STARTUP=true
export TROVE_AUTH_MODE=dev
export TROVE_AUTH_DEV_MODE_ENABLED=true
export TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT=false
export TROVE_REVIEWS_REQUIRE_APPROVAL=false
```

### Production OIDC Example (Env Vars)

```bash
export TROVE_SERVER_LISTEN=:8080
export TROVE_PUBLIC_URL=https://trove.nwks.com
export TROVE_DATABASE_URL="postgres://trove:trove@postgres:5432/trove?sslmode=require"
export TROVE_DATABASE_MIGRATE_ON_STARTUP=false
export TROVE_AUTH_MODE=oidc
export TROVE_AUTH_DEV_MODE_ENABLED=false
export TROVE_OIDC_ISSUER_URL=https://login.example.com/realms/nwks
export TROVE_OIDC_CLIENT_ID=trove
export TROVE_OIDC_CLIENT_SECRET='<client-secret>'
export TROVE_OIDC_REDIRECT_URL=https://trove.nwks.com/auth/oidc/callback
export TROVE_COOKIE_SECURE=true
```

### Production OIDC Example (Config File)

```yaml
server:
  listen: ":8080"
  publicURL: "https://trove.nwks.com"
database:
  url: "postgres://trove:trove@postgres:5432/trove?sslmode=require"
  migrateOnStartup: false
auth:
  mode: "oidc"
  devModeEnabled: false
  cookieSecure: true
oidc:
  issuerURL: "https://login.example.com/realms/nwks"
  clientID: "trove"
  clientSecret: "<client-secret>"
  redirectURL: "https://trove.nwks.com/auth/oidc/callback"
orgs:
  allowCreateOrg: false
  defaultOrg: "nwks"
packages:
  createPackageOnPush: false
  createNamespaceOnPush: false
storage:
  mode: "postgres"
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

### Server

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_SERVER_LISTEN` | `server.listen` | string | `:8080` | Address and port to listen on |
| `TROVE_PUBLIC_URL` | `server.publicURL` | string | empty | Public URL for redirects and links |

When `TROVE_PUBLIC_URL` or `server.publicURL` starts with `https://`, secure browser cookies are enabled automatically unless overridden.

### Database

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_DATABASE_URL` | `database.url` | string | required | PostgreSQL connection URL |
| `TROVE_DATABASE_MIGRATE_ON_STARTUP` | `database.migrateOnStartup` | boolean | `false` | Run embedded migrations on startup |

PostgreSQL is required. Trove does not support an in-memory database mode. Use `TROVE_DATABASE_MIGRATE_ON_STARTUP=true` or `database.migrateOnStartup: true` for local development and test-style environments only. Production should run migrations as an explicit deployment step.

### Authentication

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_AUTH_MODE` | `auth.mode` | string | `dev` | `dev` or `oidc` |
| `TROVE_AUTH_DEV_MODE_ENABLED` | `auth.devModeEnabled` | boolean | `true` | Enable dev/static auth mode |
| `TROVE_AUTH_DEV_TOKEN` | `auth.devToken` | string | empty | Optional fixed dev token |
| `TROVE_COOKIE_SECURE` | `auth.cookieSecure` | boolean | inferred from `TROVE_PUBLIC_URL` | Force secure browser cookies |
| `TROVE_OIDC_ISSUER_URL` | `oidc.issuerURL` | string | empty | OIDC provider issuer URL |
| `TROVE_OIDC_CLIENT_ID` | `oidc.clientID` | string | empty | OIDC client ID |
| `TROVE_OIDC_CLIENT_SECRET` | `oidc.clientSecret` | string | empty | OIDC client secret |
| `TROVE_OIDC_REDIRECT_URL` | `oidc.redirectURL` | string | empty | OIDC callback URL |
| _(not available)_ | `oidc.scopes` | string array | `["openid", "profile", "email"]` | OIDC scopes to request |

When `TROVE_AUTH_MODE=oidc` or `auth.mode: "oidc"`, Trove reads the provider's `.well-known/openid-configuration` document at startup.

### Storage

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_STORAGE_MODE` | `storage.mode` | string | `postgres` | Storage mode. The current runtime supports PostgreSQL storage only. |
| `TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES` | `storage.limits.maxArtifactFileBytes` | integer | `10485760` | Max size per artifact file |
| `TROVE_STORAGE_MAX_UNPACKED_PACKAGE_BYTES` | `storage.limits.maxUnpackedPackageBytes` | integer | `104857600` | Max total unpacked archive size |
| `TROVE_STORAGE_MAX_ARTIFACTS_PER_VERSION` | `storage.limits.maxArtifactsPerVersion` | integer | `1000` | Max artifact count per version |

Artifact bytes are stored in PostgreSQL `bytea` in the current implementation. The `storage.limits` keys are nested under `storage.limits` in YAML.

### Raw Artifact Access

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT` | `raw.requireAuthByDefault` | boolean | `true` | Require auth for raw URLs unless public package access is allowed |
| `TROVE_RAW_ALLOW_PUBLIC_NAMESPACES` | `raw.allowPublicNamespaces` | boolean | `true` | Allow public namespace/package raw artifact access |
| `TROVE_RAW_ALLOW_PUBLIC_PACKAGES` | `raw.allowPublicPackages` | boolean | `true` | Allow public packages to expose raw artifacts anonymously |

### Organizations

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_ALLOW_CREATE_ORG` | `orgs.allowCreateOrg` | boolean | `true` | Allow authenticated users with `org:write` to create orgs through API/UI |
| `TROVE_ORG` | `orgs.defaultOrg` | string | empty | Optional org slug to ensure exists on startup |

If `TROVE_ALLOW_CREATE_ORG=false` or `orgs.allowCreateOrg: false`, `TROVE_ORG` / `orgs.defaultOrg` is required so the instance has at least one org available.

### Packages

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_CREATE_PACKAGE_ON_PUSH` | `packages.createPackageOnPush` | boolean | `true` | Allow draft creation to auto-create a missing package when the org and namespace exist |
| `TROVE_CREATE_NAMESPACE_ON_PUSH` | `packages.createNamespaceOnPush` | boolean | `true` | Allow draft creation to auto-create a missing namespace when the org exists |

These are server-side controls. Missing orgs are never auto-created by `trove push`; create the org first or configure `orgs.defaultOrg` on the server.

### Reviews

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_REVIEWS_REQUIRE_APPROVAL` | `reviews.requireApproval` | boolean | `true` | Require approval before publish |
| `TROVE_REVIEWS_MINIMUM_APPROVALS` | `reviews.minimumApprovals` | integer | `1` | Minimum approvals required |
| `TROVE_REVIEWS_ALLOW_SELF_APPROVAL` | `reviews.allowSelfApproval` | boolean | `false` | Allow submitters to approve their own changes |

Keep self-approval disabled in shared environments. For single-user local browser workflows, set `TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true` or `reviews.allowSelfApproval: true`, then approve and publish submitted versions from `/reviews`.

### Security Scanning

| Env Variable | YAML Key | Type | Default | Description |
|---|---|---|---|---|
| `TROVE_SECURITY_SECRET_SCANNING` | `security.secretScanning` | boolean | `true` | Enable secret scanning |
| `TROVE_SECURITY_UNSAFE_INSTRUCTION_SCANNING` | `security.unsafeInstructionScanning` | boolean | `true` | Enable unsafe-instruction scanning |

### Next Steps

- Learn about [Deployment](/operations/deployment) for running Trove
