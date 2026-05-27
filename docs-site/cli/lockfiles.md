# Lockfiles

## Why

When your project installs packages from Trove, you need reproducibility. Every developer, every CI run, every agent should get the exact same artifact bytes. The lockfile (`.trove.lock.yaml`) pins exact versions and digests so that `trove install` and `trove update` are deterministic.

## How

### Full Example

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
  - package: nwks/frontend/react-defaults
    requestedSelector: latest
    version: 2.0.0
    digest: sha256:jkl012...
    installedAt: "2026-05-23T00:00:00Z"
    artifacts:
      - source: AGENTS.md
        target: AGENTS.md
        digest: sha256:mno345...
```

### Fields Reference

#### Root

| Field | Type | Description |
|---|---|---|
| `apiVersion` | string | Must be `trove.io/v1` |
| `kind` | string | Must be `TroveLock` |
| `generatedBy` | object | Tool that created this lockfile |
| `project` | object | Project metadata |
| `installs` | array | List of installed package pins |

#### generatedBy

| Field | Type | Description |
|---|---|---|
| `name` | string | Tool name (e.g., `trove`) |
| `version` | string | Tool version (e.g., `0.1.0`) |
| `generatedAt` | string | RFC 3339 timestamp of generation |

#### project

| Field | Type | Required | Description |
|---|---|---|---|
| `org` | string | Yes | Project's organization |
| `name` | string | Yes | Project name |
| `repo` | string | Yes | Repository URL |

#### installs[]

| Field | Type | Required | Description |
|---|---|---|---|
| `package` | string | Yes | Full `org/namespace/package` reference |
| `requestedSelector` | string | Yes | The selector used during install (e.g., `latest`) |
| `version` | string | Yes | Pinned exact version |
| `digest` | string | Yes | Pinned package-version digest |
| `installedAt` | string | Yes | RFC 3339 timestamp of installation |
| `artifacts` | array | Yes | Installed artifact details |

#### installs[].artifacts[]

| Field | Type | Required | Description |
|---|---|---|---|
| `source` | string | Yes | Artifact path in the package |
| `target` | string | Yes | Where the artifact was installed |
| `digest` | string | Yes | Per-artifact digest |

### Canonical Format

The CLI writes lockfiles as canonical YAML with stable ordering:

- Keys are sorted alphabetically
- Arrays maintain install order
- Timestamps use RFC 3339 UTC format
- No comments or hand formatting is preserved on rewrite

When `trove install` or `trove update --apply` modifies the lockfile, it rewrites the entire file. Any manual comments or formatting changes are lost.

### Package References

All lockfile package references must use the full `org/namespace/package` form:

```yaml
# Correct
- package: nwks/platform/agent-backend

# Incorrect — shorthand not supported
- package: agent-backend
- package: platform/agent-backend
```

### How Lockfiles are Used

| Command | Behavior |
|---|---|
| `trove check` | Reads `.trove.lock.yaml`, checks for updates |
| `trove update` | Reads `.trove.lock.yaml`, shows available updates |
| `trove update --apply` | Reads `.trove.lock.yaml`, updates packages, rewrites lockfile |

### Update Checks

When you run `trove check`, the server compares each pinned version and digest against the current selector resolution:

```
Pinned: nwks/platform/agent-backend@1.0.0 (sha256:abc123...)
@latest resolves to: 1.0.0 (sha256:abc123...)
→ Status: current

Pinned: nwks/frontend/react-defaults@2.0.0 (sha256:def456...)
@latest resolves to: 2.1.0 (sha256:ghi789...)
→ Status: update available
```

### Committing Lockfiles

Lockfiles should be committed to version control alongside your project source code. This ensures:

- Every developer gets the same artifact versions
- CI runs are reproducible
- Rollbacks are possible by reverting the lockfile

### Next Steps

- [install](/cli/install) — generates the lockfile during installation
- [check](/cli/check) — verifies lockfile pins are current
