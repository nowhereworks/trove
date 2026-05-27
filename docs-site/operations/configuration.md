# Configuration

## Why

Trove needs to know how to listen for requests, connect to the database, authenticate users, and enforce limits. All configuration is done through a single YAML file or environment variables, with sensible defaults for development.

## How

### Full Example

```yaml
server:
  listen: ":8080"
  publicUrl: "https://trove.nwks.com"

database:
  url: "postgres://trove:trove@postgres:5432/trove?sslmode=disable"
  migrateOnStartup: false

auth:
  mode: oidc
  devModeEnabled: false
  oidc:
    issuerUrl: "https://login.microsoftonline.com/tenant-id/v2.0"
    clientId: "trove"
    clientSecretRef: "TROVE_OIDC_CLIENT_SECRET"
    redirectUrl: "https://trove.nwks.com/auth/oidc/callback"
    scopes:
      - openid
      - profile
      - email
  tokens:
    hashSecretRef: "TROVE_TOKEN_HASH_SECRET"

storage:
  mode: postgres
  limits:
    maxArtifactFileBytes: 10485760
    maxUnpackedPackageBytes: 104857600
    maxArtifactsPerVersion: 1000
  s3:
    enabled: false
    endpoint: "http://rustfs:9000"
    bucket: "trove-artifacts"
    accessKeyRef: "TROVE_S3_ACCESS_KEY"
    secretKeyRef: "TROVE_S3_SECRET_KEY"

raw:
  requireAuthByDefault: true
  allowPublicNamespaces: true
  allowPublicPackages: true

orgs:
  allowCreateOrg: true
  defaultOrg: "nwks"

packages:
  createPackageOnPush: true
  createNamespaceOnPush: true

reviews:
  requireApproval: true
  minimumApprovals: 1
  allowSelfApproval: false

security:
  secretScanning: true
  unsafeInstructionScanning: true
```

### Server

| Key | Type | Default | Description |
|---|---|---|---|
| `server.listen` | string | `:8080` | Address and port to listen on |
| `server.publicUrl` | string | — | Public URL for redirects and links |

### Database

| Key | Type | Default | Description |
|---|---|---|---|
| `database.url` | string | Required | PostgreSQL connection URL |
| `database.migrateOnStartup` | boolean | `false` | Run migrations on startup (dev/tests only) |

**Note:** PostgreSQL is required. Trove does not support an in-memory database mode. `migrateOnStartup` is for local development and tests only. Production should run migrations as an explicit command or deployment job.

### Authentication

| Key | Type | Default | Description |
|---|---|---|---|
| `auth.mode` | string | `oidc` | `oidc` or `dev` |
| `auth.devModeEnabled` | boolean | `false` | Enable dev/static auth mode |
| `auth.oidc.issuerUrl` | string | — | OIDC provider issuer URL |
| `auth.oidc.clientId` | string | — | OIDC client ID |
| `auth.oidc.clientSecretRef` | string | — | Environment variable name for client secret |
| `auth.oidc.redirectUrl` | string | — | OIDC callback URL |
| `auth.oidc.scopes` | array | `[openid, profile, email]` | Requested OIDC scopes |
| `auth.tokens.hashSecretRef` | string | — | Environment variable name for token hash secret |

### Storage

| Key | Type | Default | Description |
|---|---|---|---|
| `storage.mode` | string | `postgres` | `postgres` (S3 reserved for future) |
| `storage.limits.maxArtifactFileBytes` | integer | `10485760` (10 MiB) | Max size per artifact file |
| `storage.limits.maxUnpackedPackageBytes` | integer | `104857600` (100 MiB) | Max total unpacked package size |
| `storage.limits.maxArtifactsPerVersion` | integer | `1000` | Max artifact count per version |
| `storage.s3.enabled` | boolean | `false` | Enable S3-compatible storage |
| `storage.s3.endpoint` | string | — | S3 endpoint URL |
| `storage.s3.bucket` | string | — | S3 bucket name |
| `storage.s3.accessKeyRef` | string | — | Environment variable for access key |
| `storage.s3.secretKeyRef` | string | — | Environment variable for secret key |

### Raw Artifact Access

| Key | Type | Default | Description |
|---|---|---|---|
| `raw.requireAuthByDefault` | boolean | `true` | Require auth for raw URLs |
| `raw.allowPublicNamespaces` | boolean | `true` | Allow public namespaces |
| `raw.allowPublicPackages` | boolean | `true` | Allow public packages |

### Organizations

| Key | Type | Default | Description |
|---|---|---|---|
| `orgs.allowCreateOrg` | boolean | `true` | Allow authenticated users with `org:write` to create orgs through the API and UI |
| `orgs.defaultOrg` | string | — | Optional org slug to ensure exists on startup |

Set `TROVE_ORG` when you want Trove to bootstrap a known org at startup. This does not restrict the instance to that org, and additional orgs can still be created when `TROVE_ALLOW_CREATE_ORG=true`.

If `TROVE_ALLOW_CREATE_ORG=false`, `TROVE_ORG` is required so the instance has at least one org available.

### Packages

| Key | Type | Default | Description |
|---|---|---|---|
| `packages.createPackageOnPush` | boolean | `true` | Allow `trove push` draft creation to auto-create a missing package when the org and namespace exist |
| `packages.createNamespaceOnPush` | boolean | `true` | Allow `trove push` draft creation to auto-create a missing namespace when the org exists |

These are server-side controls. The CLI does not read them. Missing orgs are never auto-created by `trove push`; use `TROVE_ORG` startup bootstrapping or the org creation API/UI first.

### Reviews

| Key | Type | Default | Description |
|---|---|---|---|
| `reviews.requireApproval` | boolean | `true` | Require approval before publishing |
| `reviews.minimumApprovals` | integer | `1` | Minimum approvals required |
| `reviews.allowSelfApproval` | boolean | `false` | Allow self-approval |

### Security

| Key | Type | Default | Description |
|---|---|---|---|
| `security.secretScanning` | boolean | `true` | Enable secret scanning |
| `security.unsafeInstructionScanning` | boolean | `true` | Enable unsafe instruction scanning |

### Environment Variables

Secrets are referenced by environment variable name:

| Variable | Used By |
|---|---|
| `TROVE_OIDC_CLIENT_SECRET` | OIDC client secret |
| `TROVE_TOKEN_HASH_SECRET` | API token hashing |
| `TROVE_S3_ACCESS_KEY` | S3 access key (if enabled) |
| `TROVE_S3_SECRET_KEY` | S3 secret key (if enabled) |
| `TROVE_DATABASE_URL` | PostgreSQL connection (alternative to config file) |
| `TROVE_ALLOW_CREATE_ORG` | Enable or disable org creation through API/UI (`true` by default) |
| `TROVE_ORG` | Optional org slug to ensure exists on startup; required when `TROVE_ALLOW_CREATE_ORG=false` |
| `TROVE_CREATE_PACKAGE_ON_PUSH` | Server-side control for package auto-creation during `trove push` (`true` by default) |
| `TROVE_CREATE_NAMESPACE_ON_PUSH` | Server-side control for namespace auto-creation during `trove push` (`true` by default) |

### Next Steps

- Learn about [Deployment](/operations/deployment) for running Trove
