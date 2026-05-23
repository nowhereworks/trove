# Adoption Dashboard

## Why

Package maintainers need to know: who is using my packages? Are teams adopting the latest version? Is a deprecated version still widely installed? The adoption dashboard provides governance visibility without exposing downstream project details to unauthorized users.

## How

### What Adoption Shows

The adoption dashboard displays:

- **Project count** — how many distinct projects have installed the package
- **Version breakdown** — how many projects use each version
- **Install timestamps** — when projects last checked or updated

### Viewing Adoption

```bash
GET /api/v1/packages/nwks/platform/agent-backend/adoption
```

Response:

```json
{
  "projectCount": 12,
  "versionCount": 3,
  "versions": [
    {
      "version": "1.0.0",
      "projectCount": 8,
      "lastInstalledAt": "2026-05-22T14:30:00Z"
    },
    {
      "version": "0.9.0",
      "projectCount": 3,
      "lastInstalledAt": "2026-05-15T09:00:00Z"
    },
    {
      "version": "0.8.0",
      "projectCount": 1,
      "lastInstalledAt": "2026-04-01T12:00:00Z"
    }
  ]
}
```

### Aggregate vs Detailed Views

| View | Who Sees It | What It Shows |
|---|---|---|
| **Aggregate** | Everyone (including anonymous for public packages) | Project counts, version counts, install timestamps |
| **Detailed** | Authorized org/package users | Project names, repository URLs, team ownership |

Public package adoption views default to aggregate counts. Project names and repository URLs are only visible to users with org or package access.

### How Adoption is Reported

Adoption data comes from CLI commands when configured with an API token:

```bash
# trove install reports adoption
trove install nwks/platform/agent-backend@stable --target . --report

# trove check reports adoption
trove check --lock .trove.lock.yaml --report

# trove update --apply reports adoption
trove update --lock .trove.lock.yaml --apply --report
```

The CLI sends a report to the server:

```bash
POST /api/v1/projects/report
Authorization: Bearer <token>

{
  "org": "nwks",
  "name": "payments-api",
  "repoUrl": "https://git.nwks.com/payments/payments-api",
  "installed": [
    {
      "package": "nwks/platform/agent-backend",
      "version": "1.0.0",
      "digest": "sha256:abc123..."
    }
  ]
}
```

### What is NOT Reported

- Artifact file contents are never sent
- Project source code is never sent
- Only lockfile summaries (package, version, digest, target paths) are reported

### Adoption UI

The Trove UI includes an adoption dashboard page showing:

- Package-level adoption counts
- Per-version breakdown with bar charts
- Trend indicators (increasing, stable, decreasing)
- Filter by package, namespace, or org

### Use Cases

| Scenario | How Adoption Helps |
|---|---|
| Deciding to deprecate a version | See how many projects still use it |
| Measuring package impact | Track adoption growth over time |
| Identifying laggards | Find projects on old versions that should update |
| Governance reporting | Show leadership which packages are widely adopted |

### Next Steps

- Learn how [Search](/discovery/search) helps discover packages
- See how [trove install](/cli/install) reports adoption data
