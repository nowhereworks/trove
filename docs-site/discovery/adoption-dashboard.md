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
      "installCount": 8
    },
    {
      "version": "0.9.0",
      "installCount": 3
    },
    {
      "version": "0.8.0",
      "installCount": 1
    }
  ]
}
```

### Aggregate Views

| View | Who Sees It | What It Shows |
|---|---|---|
| **Aggregate** | Everyone with package read access | Project counts and per-version install counts |

The current endpoint returns aggregate counts only. Project names and repository URLs are not returned by the adoption read API.

### How Adoption is Reported

Adoption data comes from project adoption reports sent to the server:

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
- Only package, version, and digest summaries are reported

### Adoption UI

The Trove UI includes an adoption dashboard page showing:

- Package-level adoption counts
- Per-version breakdown for each package card

### Use Cases

| Scenario | How Adoption Helps |
|---|---|
| Deciding to deprecate a version | See how many projects still use it |
| Measuring package impact | Track adoption growth over time |
| Identifying laggards | Find projects on old versions that should update |
| Governance reporting | Show leadership which packages are widely adopted |

### Next Steps

- Learn how [Search](/discovery/search) helps discover packages
- Use [Management APIs](/api/management-apis) to register projects and report adoption
