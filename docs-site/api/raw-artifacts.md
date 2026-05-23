# Raw Artifacts

## Why

Agents and tools need direct access to individual artifact files — `AGENTS.md`, skills, commands — without downloading the whole package archive. Raw artifact URLs serve the exact bytes of a file, with proper caching headers and digest-based ETags for conditional requests.

## How

### Exact Version URLs

```
GET /raw/{org}/{namespace}/{package}/{version}/{path...}
```

Example:

```
GET /raw/nwks/platform/agent-backend/1.0.0/AGENTS.md
```

Response:

```
HTTP/1.1 200 OK
Content-Type: text/markdown; charset=utf-8
ETag: "sha256:def456..."
Cache-Control: public, max-age=31536000, immutable

# Backend Agent Instructions
...
```

### Selector URLs (Redirect)

When you use a selector instead of an exact version, the server resolves and redirects:

```
GET /raw/nwks/platform/agent-backend/stable/AGENTS.md
```

Response:

```
HTTP/1.1 302 Found
Location: /raw/nwks/platform/agent-backend/1.0.0/AGENTS.md
Cache-Control: no-cache
```

The client follows the redirect to the exact version URL.

### Caching Behavior

| URL Type | Cache-Control | ETag |
|---|---|---|---|
| Exact version (public) | `public, max-age=31536000, immutable` | Artifact digest |
| Exact version (private) | `private, max-age=31536000, immutable` | Artifact digest |
| Selector (alias) | `no-cache` | — |

### Conditional Requests

Use the `If-None-Match` header with the ETag:

```bash
GET /raw/nwks/platform/agent-backend/1.0.0/AGENTS.md
If-None-Match: "sha256:def456..."
```

If the artifact hasn't changed:

```
HTTP/1.1 304 Not Modified
```

### Authentication

Raw URLs require authentication by default. Public packages allow anonymous reads:

| Config | Effect |
|---|---|
| `raw.requireAuthByDefault: true` | All raw URLs require auth unless package is public |
| `raw.allowPublicPackages: true` | Public packages allow anonymous raw reads |
| `raw.allowPublicNamespaces: true` | Public namespaces can contain public packages |

### Path Resolution

The `{path...}` segment matches the artifact `path` declared in the manifest:

```yaml
spec:
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
    - path: skills/backend-api/SKILL.md
      type: skill
    - path: commands/lint.md
      type: command
```

Accessible URLs:

```
/raw/nwks/platform/agent-backend/1.0.0/AGENTS.md
/raw/nwks/platform/agent-backend/1.0.0/skills/backend-api/SKILL.md
/raw/nwks/platform/agent-backend/1.0.0/commands/lint.md
```

### Error Responses

| Status | Code | Cause |
|---|---|---|
| 400 | `INVALID_ARTIFACT_PATH` | Path contains `..` or is empty |
| 401 | `UNAUTHORIZED` | Authentication required |
| 404 | `ARTIFACT_NOT_FOUND` | Path doesn't exist in version |
| 404 | `VERSION_NOT_FOUND` | Version doesn't exist |
| 404 | `PACKAGE_NOT_FOUND` | Package doesn't exist or not visible |

### Example: Fetching with curl

```bash
# With authentication
curl -H "Authorization: Bearer $TROVE_TOKEN" \
  https://trove.nwks.com/raw/nwks/platform/agent-backend/1.0.0/AGENTS.md

# Conditional request (only download if changed)
curl -H "Authorization: Bearer $TROVE_TOKEN" \
  -H 'If-None-Match: "sha256:def456..."' \
  https://trove.nwks.com/raw/nwks/platform/agent-backend/1.0.0/AGENTS.md
```

### Next Steps

- See [Archives](/api/archives) for downloading full packages
- Learn about [Public APIs](/api/public-apis) for metadata endpoints
