# trove fetch

## Why

Sometimes you need a single artifact file, not the whole package. Maybe you want to preview `AGENTS.md` before installing, or grab a specific skill to review. `trove fetch` downloads individual artifacts from a resolved package version.

## How

### Basic Usage

```bash
trove fetch nwks/platform/agent-backend@stable AGENTS.md
```

Output:

```
Fetched nwks/platform/agent-backend@1.0.0 → AGENTS.md
```

The artifact content is written to stdout by default. Redirect to a file:

```bash
trove fetch nwks/platform/agent-backend@stable AGENTS.md > AGENTS.md
```

### Output to Directory

```bash
trove fetch nwks/platform/agent-backend@stable AGENTS.md --output ./docs
```

This writes the artifact to `./docs/AGENTS.md`.

### JSON Output

```bash
trove fetch nwks/platform/agent-backend@stable AGENTS.md --json
```

Output:

```json
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "resolvedVersion": "1.0.0",
  "path": "AGENTS.md",
  "digest": "sha256:def456...",
  "sizeBytes": 4096,
  "contentType": "text/markdown; charset=utf-8"
}
```

### Fetching Different Selectors

```bash
# By stable alias
trove fetch nwks/platform/agent-backend@stable AGENTS.md

# By exact version
trove fetch nwks/platform/agent-backend@1.0.0 AGENTS.md

# By digest
trove fetch nwks/platform/agent-backend@sha256:abc123... skills/backend-api/SKILL.md
```

### What Gets Downloaded

The CLI resolves the selector to an exact version, then fetches the specific artifact path from that version. The artifact must be declared in the manifest.

### Error Cases

| Error | Cause |
|---|---|
| `ARTIFACT_NOT_FOUND` | The path doesn't exist in this version |
| `PACKAGE_NOT_FOUND` | The package doesn't exist or you lack access |
| `VERSION_NOT_FOUND` | The selector doesn't resolve |
| `UNAUTHORIZED` | Authentication is required |

### Next Steps

- [install](/cli/install) — install all required artifacts at once
- [resolve](/cli/resolve) — see what version a selector points to
