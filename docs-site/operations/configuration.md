# Configuration

## Why

Trove needs startup configuration for HTTP serving, PostgreSQL, authentication, raw artifact access, review policy, and upload limits. The current implementation reads configuration from environment variables only.

## How

### Development Example

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

### Production OIDC Example

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

### Server

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_SERVER_LISTEN` | string | `:8080` | Address and port to listen on |
| `TROVE_PUBLIC_URL` | string | empty | Public URL for redirects and links |

When `TROVE_PUBLIC_URL` starts with `https://`, secure browser cookies are enabled automatically unless overridden.

### Database

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_DATABASE_URL` | string | required | PostgreSQL connection URL |
| `TROVE_DATABASE_MIGRATE_ON_STARTUP` | boolean | `false` | Run embedded migrations on startup |

PostgreSQL is required. Trove does not support an in-memory database mode. Use `TROVE_DATABASE_MIGRATE_ON_STARTUP=true` for local development and test-style environments only. Production should run migrations as an explicit deployment step.

### Authentication

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_AUTH_MODE` | string | `dev` | `dev` or `oidc` |
| `TROVE_AUTH_DEV_MODE_ENABLED` | boolean | `true` | Enable dev/static auth mode |
| `TROVE_AUTH_DEV_TOKEN` | string | empty | Optional fixed dev token |
| `TROVE_COOKIE_SECURE` | boolean | inferred from `TROVE_PUBLIC_URL` | Force secure browser cookies |
| `TROVE_OIDC_ISSUER_URL` | string | empty | OIDC provider issuer URL |
| `TROVE_OIDC_CLIENT_ID` | string | empty | OIDC client ID |
| `TROVE_OIDC_CLIENT_SECRET` | string | empty | OIDC client secret |
| `TROVE_OIDC_REDIRECT_URL` | string | empty | OIDC callback URL |

When `TROVE_AUTH_MODE=oidc`, Trove reads the provider's `.well-known/openid-configuration` document at startup.

### Storage

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_STORAGE_MODE` | string | `postgres` | Storage mode. The current runtime supports PostgreSQL storage only. |
| `TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES` | integer | `10485760` | Max size per artifact file |
| `TROVE_STORAGE_MAX_UNPACKED_PACKAGE_BYTES` | integer | `104857600` | Max total unpacked archive size |
| `TROVE_STORAGE_MAX_ARTIFACTS_PER_VERSION` | integer | `1000` | Max artifact count per version |

Artifact bytes are stored in PostgreSQL `bytea` in the current implementation.

### Raw Artifact Access

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT` | boolean | `true` | Require auth for raw URLs unless public package access is allowed |
| `TROVE_RAW_ALLOW_PUBLIC_NAMESPACES` | boolean | `true` | Allow public namespace/package raw artifact access |
| `TROVE_RAW_ALLOW_PUBLIC_PACKAGES` | boolean | `true` | Allow public packages to expose raw artifacts anonymously |

### Organizations

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_ALLOW_CREATE_ORG` | boolean | `true` | Allow authenticated users with `org:write` to create orgs through API/UI |
| `TROVE_ORG` | string | empty | Optional org slug to ensure exists on startup |

If `TROVE_ALLOW_CREATE_ORG=false`, `TROVE_ORG` is required so the instance has at least one org available.

### Packages

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_CREATE_PACKAGE_ON_PUSH` | boolean | `true` | Allow draft creation to auto-create a missing package when the org and namespace exist |
| `TROVE_CREATE_NAMESPACE_ON_PUSH` | boolean | `true` | Allow draft creation to auto-create a missing namespace when the org exists |

These are server-side controls. Missing orgs are never auto-created by `trove push`; create the org first or configure `TROVE_ORG` on the server.

### Reviews

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_REVIEWS_REQUIRE_APPROVAL` | boolean | `true` | Require approval before publish |
| `TROVE_REVIEWS_MINIMUM_APPROVALS` | integer | `1` | Minimum approvals required |
| `TROVE_REVIEWS_ALLOW_SELF_APPROVAL` | boolean | `false` | Allow submitters to approve their own changes |

### Security Scanning

| Variable | Type | Default | Description |
|---|---|---|---|
| `TROVE_SECURITY_SECRET_SCANNING` | boolean | `true` | Enable secret scanning |
| `TROVE_SECURITY_UNSAFE_INSTRUCTION_SCANNING` | boolean | `true` | Enable unsafe-instruction scanning |

### Next Steps

- Learn about [Deployment](/operations/deployment) for running Trove
