# Public APIs

## Why

Agent-facing APIs are first-class in Trove. Agents, CLIs, and CI systems need to resolve package versions, search for packages, fetch manifests, download artifacts, and check for updates — all without a browser. These endpoints serve JSON and raw bytes, with visibility-dependent authentication.

## How

### Common Response Headers

Every response includes:

| Header | Description |
|---|---|
| `X-Request-Id` | Unique request identifier for tracing |
| `Content-Type` | `application/json` for JSON responses |

### Error Shape

All errors use a consistent JSON format:

```json
{
  "error": {
    "code": "PACKAGE_NOT_FOUND",
    "message": "Package nwks/platform/agent-backend was not found.",
    "requestId": "req_abc123..."
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Authentication is required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `PACKAGE_NOT_FOUND` | 404 | Package doesn't exist or is not visible |
| `VERSION_NOT_FOUND` | 404 | Version doesn't exist |
| `ARTIFACT_NOT_FOUND` | 404 | Artifact path doesn't exist in version |
| `INVALID_MANIFEST` | 400 | Manifest validation failed |
| `VERSION_ALREADY_EXISTS` | 409 | Version already published |
| `VERSION_IMMUTABLE` | 409 | Published version cannot be modified |
| `REVIEW_REQUIRED` | 403 | Version needs review before publishing |
| `APPROVAL_REQUIRED` | 403 | Version needs approval before publishing |
| `POLICY_VIOLATION` | 403 | Policy check failed |
| `YANKED_VERSION` | 404 | Version has been yanked |

### Pagination

List endpoints use cursor pagination:

```json
{
  "items": [...],
  "nextCursor": "eyJpZCI6IjAxOTA..."
}
```

| Param | Type | Default | Description |
|---|---|---|---|
| `limit` | integer | 50 | Results per page (server may cap) |
| `cursor` | string | — | Opaque cursor from previous response |

### Endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/resolve/{org}/{namespace}/{package}@{selector}` | visibility-dependent | Resolve a selector to an exact version |
| `GET` | `/api/v1/search/packages` | visibility-dependent | Search discoverable packages |
| `GET` | `/api/v1/packages` | visibility-dependent | List packages (paginated) |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}` | visibility-dependent | Get package metadata |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/manifest` | visibility-dependent | Get the package manifest |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.tar.gz` | visibility-dependent | Download package archive |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.zip` | visibility-dependent | Download package archive |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/adoption` | visibility-dependent | Get adoption statistics |
| `GET` | `/raw/{org}/{namespace}/{package}/{version}/{path...}` | visibility-dependent | Fetch one artifact file |
| `GET` | `/raw/{org}/{namespace}/{package}/{selector}/{path...}` | visibility-dependent | Resolve selector, then redirect or serve |
| `POST` | `/api/v1/updates/check` | token/session | Check for newer compatible releases |
| `POST` | `/api/v1/compatibility/check` | token/session | Check package compatibility |
| `POST` | `/api/v1/projects/report` | token/session | Report project adoption |

### Resolve Response

```json
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "selector": "stable",
  "resolvedVersion": "1.0.0",
  "digest": "sha256:abc123...",
  "manifestUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/manifest",
  "archiveUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive.tar.gz"
}
```

### Package Metadata Response

```json
{
  "org": "nwks",
  "namespace": "platform",
  "name": "agent-backend",
  "displayName": "Backend Agent Defaults",
  "description": "Default agent instructions for backend services.",
  "visibility": "public",
  "lifecycle": "active",
  "latestVersion": "1.0.0",
  "stableVersion": "1.0.0",
  "maintainers": [{ "team": "platform-engineering" }],
  "labels": { "language": "golang", "framework": "chi" },
  "createdAt": "2026-01-15T00:00:00Z",
  "updatedAt": "2026-05-23T00:00:00Z"
}
```

### Next Steps

- See [Management APIs](/api/management-apis) for write endpoints
- Learn about [Raw Artifacts](/api/raw-artifacts) for direct file access
