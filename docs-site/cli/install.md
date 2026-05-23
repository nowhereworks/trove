# trove install

## Why

You've found the package you need. Now you want its artifacts installed into your project, with exact versions pinned in a lockfile so your team and CI always get the same content. `trove install` does both in one command.

## How

### Basic Usage

```bash
trove install nwks/platform/agent-backend@stable --target .
```

Output:

```
Resolving nwks/platform/agent-backend@stable → 1.0.0
Installing required artifacts:
  ✓ AGENTS.md → AGENTS.md (4096 bytes)
  ✓ skills/backend-api/SKILL.md → .opencode/skills/backend-api/SKILL.md (2048 bytes)
Writing .trove.lock.yaml
Installed nwks/platform/agent-backend@1.0.0 to .
```

### What Gets Installed

By default, `trove install` installs only artifacts marked `required: true` in the manifest:

```yaml
spec:
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true        # ← installed
    - path: skills/backend-api/SKILL.md
      type: skill
      required: false       # ← skipped unless explicitly selected
```

### Installing Optional Artifacts

To include optional artifacts, use the `--all` flag:

```bash
trove install nwks/platform/agent-backend@stable --target . --all
```

### Overwriting Existing Files

If a target file already exists with different content, `trove install` fails:

```bash
trove install nwks/platform/agent-backend@stable --target .
# ERROR: AGENTS.md already exists and differs from fetched artifact.
# Use --overwrite to replace it.
```

To force overwrite:

```bash
trove install nwks/platform/agent-backend@stable --target . --overwrite
```

### Lockfile Generation

After installing artifacts, the CLI writes `.trove.lock.yaml`:

```yaml
apiVersion: trove.io/v1
kind: TroveLock
generatedBy:
  name: trove
  version: 0.1.0
  generatedAt: "2026-05-23T00:00:00Z"
project:
  org: nwks
  name: payments-api
  repo: https://git.nwks.com/payments/payments-api
installs:
  - package: nwks/platform/agent-backend
    requestedSelector: stable
    version: 1.0.0
    digest: sha256:abc123...
    installedAt: "2026-05-23T00:00:00Z"
    artifacts:
      - source: AGENTS.md
        target: AGENTS.md
        digest: sha256:def456...
      - source: skills/backend-api/SKILL.md
        target: .opencode/skills/backend-api/SKILL.md
        digest: sha256:ghi789...
```

### Adoption Reporting

When configured with an API token, `trove install` reports adoption:

```bash
trove install nwks/platform/agent-backend@stable --target . --report
```

This sends the lockfile summary to the server so the adoption dashboard stays current.

### JSON Output

```bash
trove install nwks/platform/agent-backend@stable --target . --json
```

Output:

```json
{
  "package": "nwks/platform/agent-backend",
  "resolvedVersion": "1.0.0",
  "digest": "sha256:abc123...",
  "artifactsInstalled": [
    { "source": "AGENTS.md", "target": "AGENTS.md", "bytes": 4096 },
    { "source": "skills/backend-api/SKILL.md", "target": ".opencode/skills/backend-api/SKILL.md", "bytes": 2048 }
  ],
  "lockfile": ".trove.lock.yaml"
}
```

### Error Cases

| Error | Cause |
|---|---|
| `PACKAGE_NOT_FOUND` | Package doesn't exist or you lack access |
| `VERSION_NOT_FOUND` | Selector doesn't resolve |
| File conflict | Target file exists with different content (use `--overwrite`) |
| `UNAUTHORIZED` | Authentication required |

### Next Steps

- [check](/cli/check) — verify your installed versions are current
- [Lockfiles](/cli/lockfiles) — understand the lockfile format
