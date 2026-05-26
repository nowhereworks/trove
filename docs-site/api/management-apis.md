# Management APIs

## Why

Package lifecycle operations — creating drafts, uploading artifacts, submitting for review, approving, publishing, deprecating, and yanking — require authenticated write access. These endpoints manage the full lifecycle from creation to retirement.

## How

### Authentication

All management endpoints require authentication via session or API token. Specific endpoints require specific scopes (see [RBAC & Scopes](/security/rbac-scopes)).

### Endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/config` | none | Read public UI configuration |
| `POST` | `/api/v1/orgs` | `org:write` | Create organization |
| `POST` | `/api/v1/orgs/{org}/namespaces` | `namespace:write` | Create namespace |
| `POST` | `/api/v1/packages` | `package:write` | Create package |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions` | `package:write` | Create draft version |
| `PUT` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/{path...}` | `package:write` | Upload draft artifact |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive` | `package:write` | Upload archive (.tar.gz/.zip) |
| `POST` | `/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/submit` | `review:write` | Submit for review |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/reset-draft` | `package:write` | Reset an unpublished review version to draft |
| `POST` | `/api/v1/reviews/{reviewId}/approve` | `review:write` | Approve version |
| `POST` | `/api/v1/reviews/{reviewId}/request-changes` | `review:write` | Request changes |
| `POST` | `/api/v1/reviews/{reviewId}/comments` | `review:write` | Add review comment |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish` | `version:publish` | Publish immutable version |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/deprecate` | `package:write` | Deprecate version |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/yank` | `package:write` | Yank version |
| `PATCH` | `/api/v1/packages/{org}/{namespace}/{package}/visibility` | `package:write` | Update package visibility |
| `POST` | `/api/v1/projects` | token/session | Register a project |

### Response Codes

- **201 Created** — for create endpoints (org, namespace, package, draft version, project)
- **200 OK** — for state transition endpoints (submit, reset, approve, publish, deprecate, yank)

Create endpoints return the created resource JSON and a `Location` header.

### Public UI Configuration

```bash
GET /api/v1/config
```

Response: `200 OK`

```json
{
  "org": "nwks",
  "allowCreateOrg": true
}
```

The `org` value is the optional startup org configured with `TROVE_ORG`. The `allowCreateOrg` value mirrors `TROVE_ALLOW_CREATE_ORG` and lets the UI hide or disable org creation controls.

### Create Organization

```bash
POST /api/v1/orgs
Authorization: Bearer <token>
Content-Type: application/json

{
  "slug": "nwks",
  "displayName": "Nowhereworks",
  "visibility": "private"
}
```

Response: `201 Created`

```json
{
  "id": "org-abc123",
  "slug": "nwks",
  "displayName": "Nowhereworks",
  "visibility": "private",
  "createdAt": "2026-05-23T00:00:00Z"
}
```

If `TROVE_ALLOW_CREATE_ORG=false`, the endpoint returns `403 Forbidden`:

```json
{
  "error": {
    "code": "ORG_CREATION_DISABLED",
    "message": "Organization creation is disabled. Configure TROVE_ORG at startup or enable TROVE_ALLOW_CREATE_ORG.",
    "requestId": "req_abc123"
  }
}
```

### Create Namespace

```bash
POST /api/v1/orgs/nwks/namespaces
Authorization: Bearer <token>
Content-Type: application/json

{
  "slug": "platform",
  "displayName": "Platform Engineering",
  "visibility": "private"
}
```

Response: `201 Created`

### Create Package

```bash
POST /api/v1/packages
Authorization: Bearer <token>
Content-Type: application/json

{
  "org": "nwks",
  "namespace": "platform",
  "name": "agent-backend",
  "displayName": "Backend Agent Defaults",
  "description": "Default agent instructions for backend services.",
  "visibility": "private"
}
```

Response: `201 Created`

### Create Draft Version

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions
Authorization: Bearer <token>
Content-Type: application/json

{
  "version": "1.0.0"
}
```

Response: `201 Created`

### Update Package Visibility

```bash
PATCH /api/v1/packages/nwks/platform/agent-backend/visibility
Authorization: Bearer <token>
Content-Type: application/json

{
  "visibility": "public"
}
```

Response: `200 OK`

Returns the updated package resource. Visibility must be `private`, `internal`, or `public`.

### Upload Artifact

```bash
PUT /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/artifacts/trove.yaml
Authorization: Bearer <token>
Content-Type: application/yaml

apiVersion: trove.io/v1
kind: AgentArtifactPackage
# ... manifest content ...
```

Response: `200 OK`

### Upload Archive

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive
Authorization: Bearer <token>
Content-Type: application/gzip

<binary: package.tar.gz>
```

Response: `200 OK`

### Reset Unpublished Version

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/reset-draft
Authorization: Bearer <token>
```

Response: `200 OK`

```json
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "version": "1.0.0",
  "lifecycle": "draft"
}
```

Reset is only valid for unpublished `draft` or `review` versions. It clears review state and approvals. Published, deprecated, and yanked versions remain immutable and must be replaced with a new SemVer version.

### Publish Version

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/publish
Authorization: Bearer <token-with-version:publish>
```

Response: `200 OK`

### Deprecate Version

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/deprecate
Authorization: Bearer <token>
```

Response: `200 OK`

### Yank Version

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/yank
Authorization: Bearer <token>
```

Response: `200 OK`

### Register Project

```bash
POST /api/v1/projects
Authorization: Bearer <token>
Content-Type: application/json

{
  "org": "nwks",
  "name": "payments-api",
  "repoUrl": "https://git.nwks.com/payments/payments-api"
}
```

Response: `201 Created`

```json
{
  "id": "proj-abc123",
  "org": "nwks",
  "name": "payments-api",
  "repoUrl": "https://git.nwks.com/payments/payments-api",
  "createdAt": "2026-05-23T00:00:00Z"
}
```

### Next Steps

- See [Public APIs](/api/public-apis) for read endpoints
- Learn about [Raw Artifacts](/api/raw-artifacts) for direct file access
