# Trove Helm Chart

This chart deploys the Trove application to Kubernetes. PostgreSQL is external and must already exist.

Published chart releases are available from GitHub Container Registry:

```bash
helm install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

Use the local chart while developing chart changes:

```bash
helm install trove ./charts/trove \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

## Production Example

Prefer existing Kubernetes Secrets for credentials:

```bash
kubectl create secret generic trove-database \
  --from-literal=TROVE_DATABASE_URL='postgres://trove:trove@postgres.example:5432/trove?sslmode=require'

kubectl create secret generic trove-oidc \
  --from-literal=TROVE_OIDC_CLIENT_SECRET='<oidc-client-secret>'
```

Example values file:

```yaml
image:
  repository: nowhereworks/trove
  tag: "0.1.0"

config:
  publicUrl: https://trove.nwks.com

database:
  migrateOnStartup: false
  existingSecret:
    name: trove-database
    key: TROVE_DATABASE_URL

auth:
  mode: oidc
  devModeEnabled: false

oidc:
  issuerUrl: https://login.example.com/realms/nwks
  clientId: trove
  clientSecretExistingSecret:
    name: trove-oidc
    key: TROVE_OIDC_CLIENT_SECRET
  redirectUrl: https://trove.nwks.com/auth/oidc/callback

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: trove.nwks.com
      paths:
        - path: /
          pathType: Prefix
```

Install or upgrade with the values file:

```bash
helm upgrade --install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --values values.production.yaml
```

Run migrations explicitly in production before starting or rolling pods:

```bash
trove migrate --database-url "$DATABASE_URL"
```

## Common Values

| Value | Default | Purpose |
|---|---|---|
| `replicaCount` | `1` | Number of Trove pods |
| `image.repository` | `nowhereworks/trove` | Trove container image repository |
| `image.tag` | `latest` | Trove container image tag |
| `database.url` | `""` | Inline PostgreSQL URL for quick installs |
| `database.existingSecret.name` | `""` | Existing Secret containing `TROVE_DATABASE_URL` |
| `database.migrateOnStartup` | `false` | Run migrations on startup for dev/test style installs |
| `auth.mode` | `dev` | Auth mode, usually `oidc` for production |
| `auth.devModeEnabled` | `true` | Enable dev/static auth mode |
| `oidc.*` | `""` | OIDC issuer, client, secret, and redirect settings |
| `ingress.enabled` | `false` | Create a Kubernetes Ingress |
| `gateway.enabled` | `false` | Create a Gateway API `HTTPRoute` |
| `resources` | `{}` | Pod resource requests and limits |
| `extraEnv`, `extraEnvFrom` | `[]` | Additional environment variables |

See the full Helm documentation at https://nowhereworks.github.io/trove/docs/operations/helm-chart.html.
