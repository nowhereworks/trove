# Helm Chart

## Why

Trove publishes a Helm chart for Kubernetes installs. The chart deploys the Trove application only. PostgreSQL is intentionally external, so you can use your cluster standard database operator, cloud database, or existing PostgreSQL instance.

Published chart releases are available from GitHub Container Registry at `oci://ghcr.io/nowhereworks/charts/trove`.

## How

### Prerequisites

- Helm 3 with OCI registry support
- Kubernetes cluster access
- External PostgreSQL database
- Kubernetes Secret access for production credentials

### Install From GHCR

Use the GHCR-published chart for normal installs:

```bash
helm install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

`database.migrateOnStartup=true` is useful for quick development-style installs. Production should run migrations explicitly before starting or rolling Trove pods.

### Install From A Local Checkout

Use the local chart while developing chart changes:

```bash
helm install trove ./charts/trove \
  --set database.url='postgres://trove:trove@postgres.example:5432/trove?sslmode=require' \
  --set database.migrateOnStartup=true
```

### Upgrade

Upgrade a GHCR-installed release by changing the chart version and, when needed, the Trove image tag:

```bash
helm upgrade trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --set image.tag=0.1.0 \
  --reuse-values
```

If you need to change configuration, prefer a checked-in values file:

```bash
helm upgrade --install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --values values.production.yaml
```

### Production Values File

Use existing Kubernetes Secrets for sensitive values:

```bash
kubectl create secret generic trove-database \
  --from-literal=TROVE_DATABASE_URL='postgres://trove:trove@postgres.example:5432/trove?sslmode=require'

kubectl create secret generic trove-oidc \
  --from-literal=TROVE_OIDC_CLIENT_SECRET='<oidc-client-secret>'
```

Example `values.production.yaml`:

```yaml
image:
  repository: nowhereworks/trove
  tag: "0.1.0"
  pullPolicy: IfNotPresent

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
  tls:
    - secretName: trove-tls
      hosts:
        - trove.nwks.com

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    memory: 512Mi
```

Install it:

```bash
helm upgrade --install trove oci://ghcr.io/nowhereworks/charts/trove \
  --version 0.1.0 \
  --values values.production.yaml
```

### Gateway API

Enable Gateway API `HTTPRoute` resources instead of Ingress when your cluster uses Gateway API:

```yaml
config:
  publicUrl: https://trove.nwks.com

database:
  existingSecret:
    name: trove-database

gateway:
  enabled: true
  parentRefs:
    - name: public-gateway
  hostnames:
    - trove.nwks.com
```

Leave both `ingress.enabled` and `gateway.enabled` disabled for internal-only deployments. Enable only one external routing option for a release unless you intentionally want both resources.

### Migrations

The chart can set `TROVE_DATABASE_MIGRATE_ON_STARTUP` through `database.migrateOnStartup`, but production deployments should run migrations explicitly:

```bash
trove migrate --database-url "$DATABASE_URL"
```

Run the migration command as a controlled deployment step or Kubernetes Job before upgrading application pods.

### Chart Versioning

The chart version is stored in `charts/trove/Chart.yaml` as `version` and follows SemVer independently from Trove's application version. `appVersion` records the related Trove application version for display and tooling.

Chart releases in GHCR are immutable. Bump `version` in `charts/trove/Chart.yaml` before publishing changed chart content.

## Reference

### Required Configuration

| Value | Type | Default | Description |
|---|---|---|---|
| `database.url` | string | `""` | PostgreSQL URL. Used when `database.existingSecret.name` is empty. |
| `database.existingSecret.name` | string | `""` | Existing Secret containing the PostgreSQL URL. Preferred for production. |
| `database.existingSecret.key` | string | `TROVE_DATABASE_URL` | Secret key used when `database.existingSecret.name` is set. |
| `config.publicUrl` | string | `""` | Public URL used for redirects and links. Required for externally exposed OIDC installs. |

### Image And Release

| Value | Type | Default | Description |
|---|---|---|---|
| `replicaCount` | integer | `1` | Number of Trove pods. |
| `image.repository` | string | `nowhereworks/trove` | Trove container image repository. |
| `image.tag` | string | `latest` | Trove container image tag. Use an exact version in production. |
| `image.pullPolicy` | string | `IfNotPresent` | Kubernetes image pull policy. |
| `imagePullSecrets` | array | `[]` | Image pull secrets for private registries. |
| `nameOverride` | string | `""` | Override chart name. |
| `fullnameOverride` | string | `""` | Override generated full resource name. |

### Service Account And Pod Metadata

| Value | Type | Default | Description |
|---|---|---|---|
| `serviceAccount.create` | boolean | `true` | Create a ServiceAccount for Trove. |
| `serviceAccount.annotations` | object | `{}` | ServiceAccount annotations. |
| `serviceAccount.name` | string | `""` | Existing or explicit ServiceAccount name. |
| `podAnnotations` | object | `{}` | Annotations added to pods. |
| `podLabels` | object | `{}` | Labels added to pods. |

### Security Contexts

| Value | Type | Default | Description |
|---|---|---|---|
| `podSecurityContext.runAsUser` | integer | `10001` | Pod-level user ID. |
| `podSecurityContext.runAsGroup` | integer | `10001` | Pod-level group ID. |
| `podSecurityContext.runAsNonRoot` | boolean | `true` | Require non-root pod execution. |
| `podSecurityContext.seccompProfile.type` | string | `RuntimeDefault` | Pod seccomp profile. |
| `securityContext.allowPrivilegeEscalation` | boolean | `false` | Disable container privilege escalation. |
| `securityContext.capabilities.drop` | array | `[ALL]` | Linux capabilities to drop. |
| `securityContext.readOnlyRootFilesystem` | boolean | `true` | Run with a read-only root filesystem. |
| `securityContext.runAsUser` | integer | `10001` | Container-level user ID. |
| `securityContext.runAsGroup` | integer | `10001` | Container-level group ID. |
| `securityContext.runAsNonRoot` | boolean | `true` | Require non-root container execution. |

### Service And Routing

| Value | Type | Default | Description |
|---|---|---|---|
| `service.type` | string | `ClusterIP` | Kubernetes Service type. |
| `service.port` | integer | `80` | Service port. |
| `service.targetPort` | integer | `8080` | Container port exposed by Trove. |
| `service.annotations` | object | `{}` | Service annotations. |
| `ingress.enabled` | boolean | `false` | Create an Ingress. |
| `ingress.className` | string | `""` | Ingress class name. |
| `ingress.annotations` | object | `{}` | Ingress annotations. |
| `ingress.hosts` | array | `trove.local` example | Ingress hosts and paths. |
| `ingress.tls` | array | `[]` | Ingress TLS entries. |
| `gateway.enabled` | boolean | `false` | Create a Gateway API `HTTPRoute`. |
| `gateway.annotations` | object | `{}` | `HTTPRoute` annotations. |
| `gateway.parentRefs` | array | `[]` | Gateway parent references. Required when Gateway is enabled. |
| `gateway.hostnames` | array | `[]` | `HTTPRoute` hostnames. |
| `gateway.sectionName` | string | `""` | Optional Gateway listener section name. |

### Trove Runtime Config

| Value | Type | Default | Description |
|---|---|---|---|
| `config.serverListen` | string | `:8080` | Address Trove listens on inside the container. |
| `config.publicUrl` | string | `""` | Public URL for redirects and links. |
| `config.storageMode` | string | `postgres` | Storage mode passed to Trove. |
| `database.migrateOnStartup` | boolean | `false` | Run migrations when the server starts. Use only for dev/test style installs. |

### Authentication And OIDC

| Value | Type | Default | Description |
|---|---|---|---|
| `auth.mode` | string | `dev` | Authentication mode, usually `oidc` for production or `dev` for local testing. |
| `auth.devModeEnabled` | boolean | `true` | Enable dev/static auth mode. Disable for OIDC production installs. |
| `auth.devToken` | string | `""` | Inline dev token. Avoid inline secrets in production. |
| `auth.devTokenExistingSecret.name` | string | `""` | Existing Secret containing the dev token. |
| `auth.devTokenExistingSecret.key` | string | `TROVE_AUTH_DEV_TOKEN` | Secret key for the dev token. |
| `oidc.issuerUrl` | string | `""` | OIDC provider issuer URL. |
| `oidc.clientId` | string | `""` | OIDC client ID. |
| `oidc.clientSecret` | string | `""` | Inline OIDC client secret. Avoid inline secrets in production. |
| `oidc.clientSecretExistingSecret.name` | string | `""` | Existing Secret containing the OIDC client secret. |
| `oidc.clientSecretExistingSecret.key` | string | `TROVE_OIDC_CLIENT_SECRET` | Secret key for the OIDC client secret. |
| `oidc.redirectUrl` | string | `""` | OIDC callback URL. |

### Access, Reviews, And Scanning

| Value | Type | Default | Description |
|---|---|---|---|
| `raw.requireAuthByDefault` | boolean | `true` | Require auth for raw artifact URLs by default. |
| `raw.allowPublicNamespaces` | boolean | `true` | Allow public namespaces to expose raw artifacts. |
| `raw.allowPublicPackages` | boolean | `true` | Allow public packages to expose raw artifacts. |
| `reviews.requireApproval` | boolean | `true` | Require human approval before publish. |
| `reviews.minimumApprovals` | integer | `1` | Minimum approvals required before publish. |
| `reviews.allowSelfApproval` | boolean | `false` | Allow submitters to approve their own changes. |
| `security.secretScanning` | boolean | `true` | Enable secret scanning. |
| `security.unsafeInstructionScanning` | boolean | `true` | Enable unsafe-instruction scanning. |

### Storage Limits

| Value | Type | Default | Description |
|---|---|---|---|
| `storage.limits.maxArtifactFileBytes` | integer | `10485760` | Maximum size per artifact file. |
| `storage.limits.maxUnpackedPackageBytes` | integer | `104857600` | Maximum total unpacked package size. |
| `storage.limits.maxArtifactsPerVersion` | integer | `1000` | Maximum artifact count per version. |

### Probes And Scheduling

| Value | Type | Default | Description |
|---|---|---|---|
| `livenessProbe.enabled` | boolean | `true` | Enable liveness probe. |
| `livenessProbe.path` | string | `/healthz` | Liveness probe HTTP path. |
| `livenessProbe.initialDelaySeconds` | integer | `5` | Liveness initial delay. |
| `livenessProbe.periodSeconds` | integer | `10` | Liveness period. |
| `livenessProbe.timeoutSeconds` | integer | `1` | Liveness timeout. |
| `livenessProbe.failureThreshold` | integer | `3` | Liveness failure threshold. |
| `readinessProbe.enabled` | boolean | `true` | Enable readiness probe. |
| `readinessProbe.path` | string | `/readyz` | Readiness probe HTTP path. |
| `readinessProbe.initialDelaySeconds` | integer | `10` | Readiness initial delay. |
| `readinessProbe.periodSeconds` | integer | `5` | Readiness period. |
| `readinessProbe.timeoutSeconds` | integer | `1` | Readiness timeout. |
| `readinessProbe.failureThreshold` | integer | `3` | Readiness failure threshold. |
| `resources` | object | `{}` | Pod resource requests and limits. |
| `nodeSelector` | object | `{}` | Node selector for scheduling. |
| `tolerations` | array | `[]` | Pod tolerations. |
| `affinity` | object | `{}` | Pod affinity rules. |
| `topologySpreadConstraints` | array | `[]` | Topology spread constraints. |

### Extension Hooks

| Value | Type | Default | Description |
|---|---|---|---|
| `extraEnv` | array | `[]` | Additional environment variables for the Trove container. |
| `extraEnvFrom` | array | `[]` | Additional `envFrom` entries for the Trove container. |

## Next

- See [Deployment](/operations/deployment) for broader production deployment guidance.
- See [Configuration](/operations/configuration) for Trove application configuration.
- See [Authentication](/security/authentication) before exposing Trove outside the cluster.
