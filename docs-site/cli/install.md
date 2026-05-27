# trove install

## Why

You've found the package you need. Now you want its artifacts installed into your project, with exact versions pinned in a lockfile so your team and CI always get the same content. `trove install` does both in one command.

## How

### Basic Usage

```bash
trove install nwks/platform/agent-backend@latest --output .
```

Output:

```
Resolving nwks/platform/agent-backend@latest → 1.0.0
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

To include optional artifacts, use the `--optional` flag:

```bash
trove install nwks/platform/agent-backend@latest --output . --optional
```

### Overwriting Existing Files

If a target file already exists with different content, `trove install` fails:

```bash
trove install nwks/platform/agent-backend@latest --output .
# ERROR: AGENTS.md already exists and differs from fetched artifact.
# Use --overwrite to replace it.
```

To force overwrite:

```bash
trove install nwks/platform/agent-backend@latest --output . --overwrite
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
    requestedSelector: latest
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

### JSON Output

```bash
trove install nwks/platform/agent-backend@latest --output . --json
```

Output:

```json
{
  "package": "nwks/platform/agent-backend",
  "version": "1.0.0",
  "digest": "sha256:abc123...",
  "artifacts": 2
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
