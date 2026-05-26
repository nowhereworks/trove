# trove resolve

## Why

You know you want `nwks/platform/agent-backend`, but which version? The `@latest` alias? A specific digest? `trove resolve` tells you the exact version and digest that a selector points to, so you can verify before fetching or installing.

## How

### Basic Usage

```bash
trove resolve nwks/platform/agent-backend@latest
```

Output:

```
nwks/platform/agent-backend@latest → 2.1.0 (sha256:abc123...)
Manifest: /api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/manifest
Archive:  /api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/archive.tar.gz
```

### JSON Output

For agents and CI scripts:

```bash
trove resolve nwks/platform/agent-backend@latest --json
```

Output:

```json
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "selector": "latest",
  "resolvedVersion": "2.1.0",
  "digest": "sha256:abc123...",
  "manifestUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/manifest",
  "archiveUrl": "/api/v1/packages/nwks/platform/agent-backend/versions/2.1.0/archive.tar.gz"
}
```

### Resolving Different Selectors

```bash
# Latest version
trove resolve nwks/platform/agent-backend@latest

# Exact version (no resolution needed, but validates it exists)
trove resolve nwks/platform/agent-backend@1.0.0

# By digest
trove resolve nwks/platform/agent-backend@sha256:abc123...

# Major version selector
trove resolve nwks/platform/agent-backend@v1

# Minor version selector
trove resolve nwks/platform/agent-backend@v1.2
```

### Configuration

The CLI needs to know where your Trove server is:

```bash
# Via environment variable
export TROVE_SERVER_URL=https://trove.nwks.com

# Or via config file (~/.trove/config.yaml)
server:
  url: https://trove.nwks.com
  token: <your-api-token>
```

### Error Cases

| Error | Cause |
|---|---|
| `PACKAGE_NOT_FOUND` | The package doesn't exist or you lack access |
| `VERSION_NOT_FOUND` | The selector doesn't resolve to any version |
| `INVALID_SELECTOR` | The selector format is invalid |
| `UNAUTHORIZED` | Authentication is required for this package |

### Next Steps

- [download](/cli/download) — download individual artifacts
- [install](/cli/install) — install required artifacts and pin versions
