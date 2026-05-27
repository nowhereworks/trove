# Archives

## Why

Sometimes you need the entire package — all artifacts at once. Archives let you download a complete package version as a `.tar.gz` or `.zip` file. Archives are generated on demand from stored artifact rows and blobs, so there's no duplicate storage.

## How

### Downloading Archives

```
GET /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.tar.gz
GET /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.zip
```

Examples:

```bash
# tar.gz
curl -O https://trove.nwks.com/api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive.tar.gz

# zip
curl -O https://trove.nwks.com/api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive.zip
```

### Selector Redirects

Using a selector redirects to the exact version:

```bash
GET /api/v1/packages/nwks/platform/agent-backend/versions/latest/archive.tar.gz
```

Response:

```
HTTP/1.1 302 Found
Location: /api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/archive.tar.gz
Cache-Control: no-cache
```

### Archive Contents

The archive contains all artifacts from the package version, with entry paths matching the manifest `path` field:

```
package.tar.gz
├── Trovefile
├── AGENTS.md
├── skills/backend-api/SKILL.md
├── commands/lint.md
└── policies/go-routing.yaml
```

### Deterministic Ordering

Archive entries are sorted by artifact path for deterministic output. The same package version always produces the same archive bytes.

### Caching

| URL Type | Cache-Control | ETag |
|---|---|---|
| Exact version | `public, max-age=31536000, immutable` (public) or `private, max-age=31536000, immutable` (private) | Package digest |
| Selector | `no-cache` | — |

The ETag is based on the package digest, so any change to the manifest or artifacts produces a different ETag.

### On-Demand Generation

Archives are generated on demand from artifact rows and blob content. The generation process:

1. Resolves the version
2. Lists all artifacts for the version
3. Sorts artifacts by path
4. Creates the archive in memory
5. Returns the archive bytes

Archive generation does not mutate package version state.

### Supported Formats

| Format | Content-Type | Extension |
|---|---|---|
| gzip-compressed tar | `application/gzip` | `.tar.gz` |
| ZIP | `application/zip` | `.zip` |

### Authentication

Archive endpoints follow the same visibility rules as other API endpoints:

- Private packages require authentication
- Public packages allow anonymous downloads
- Selector redirects preserve auth requirements

### Error Responses

| Status | Code | Cause |
|---|---|---|
| 400 | `INVALID_ARCHIVE` | Archive format is invalid (for uploads) |
| 401 | `UNAUTHORIZED` | Authentication required |
| 404 | `VERSION_NOT_FOUND` | Version doesn't exist |
| 404 | `PACKAGE_NOT_FOUND` | Package doesn't exist or not visible |

### Uploading Archives

You can also upload archives to populate a draft version:

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive
Authorization: Bearer <token>
Content-Type: application/gzip

<binary: package.tar.gz>
```

The server extracts the archive, validates paths, and stores individual artifact files. Supported upload formats are `.tar.gz` and `.zip`.

### Next Steps

- See [Raw Artifacts](/api/raw-artifacts) for individual file access
- Learn about the [Upload & Publish Flow](/publishing/upload-publish-flow)
