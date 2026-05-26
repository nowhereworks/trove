# Version Selectors

## Why

Users and agents rarely know the exact version number they want. They want "the latest release" or "the newest v2 patch." Selectors let you reference packages flexibly while the server resolves them to exact, immutable versions before serving content.

## How

### Selector Types

| Selector | Resolution | Example |
|---|---|---|---|
| `@latest` | Newest published version | `nwks/platform/agent-backend@latest` → `2.1.0` |
| `@sha256:<digest>` | Exact package-version digest | `nwks/platform/agent-backend@sha256:abc123...` |
| `@v3` | Newest version with major `3` | `nwks/platform/agent-backend@v3` → `3.4.1` |
| `@v3.2` | Newest patch in minor `3.2` | `nwks/platform/agent-backend@v3.2` → `3.2.5` |

### Resolution Flow

When you use a selector, the server:

1. Parses the selector type (alias, digest, or derived major/minor)
2. Resolves the alias or scans versions for derived selectors
3. Returns the exact immutable version number
4. Serves content from that exact version

```
Request: GET /api/v1/resolve/nwks/platform/agent-backend@latest
Response:
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "selector": "latest",
  "resolvedVersion": "2.1.0",
  "digest": "sha256:def456...",
  "manifestUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/manifest",
  "archiveUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/archive.tar.gz"
}
```

### Alias Redirects

Selector-based raw URLs redirect to exact version URLs:

```
GET /raw/nwks/platform/agent-backend/AGENTS.md@latest
→ 302 Found → /raw/nwks/platform/agent-backend/AGENTS.md@2.1.0
```

This keeps caching behavior explicit — alias responses use `Cache-Control: no-cache`, while exact version responses use immutable caching.

### Selector Rules

- `@latest` is the only named alias
- `@v3` and `@v3.2` are derived from SemVer
- `@sha256:<digest>` identifies a whole package version, not individual files
- Yanked versions are hidden from `@latest` but remain fetchable by exact version for authorized clients
- Prerelease versions are not supported in the MVP

### Next Steps

- Understand [Lifecycle States](/concepts/lifecycle-states) that determine which versions selectors can resolve
- Learn how to [resolve](/cli/resolve) selectors from the CLI
