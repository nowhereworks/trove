# trove check

## Why

Your project has installed packages pinned in `.trove.lock.yaml`. Are those versions still current? Has a version been yanked? Is there a newer release? `trove check` answers these questions without changing anything — it's advisory by default, perfect for CI.

## How

### Basic Usage

```bash
trove check --lock .trove.lock.yaml
```

Output:

```
Checking .trove.lock.yaml...

nwks/platform/agent-backend@1.0.0
  Status: current
  Latest: 1.0.0

nwks/frontend/react-defaults@2.0.0
  Status: update available
  Latest: 2.1.0

All checks passed. 1 package has updates available.
```

### Exit Codes

| Exit Code | Meaning |
|---|---|
| `0` | All checks passed |
| `1` | Yanked versions detected |
| `2` | Error (lockfile not found, server unreachable, etc.) |

### CI Integration

```bash
# Advisory mode (default) — exits 0 even if updates are available
trove check --lock .trove.lock.yaml

# Strict mode — exit non-zero for any update
trove check --lock .trove.lock.yaml --fail-on-update

# Fail only on yanked versions
trove check --lock .trove.lock.yaml --fail-on-yanked
```

### JSON Output

```bash
trove check --lock .trove.lock.yaml --json
```

Output:

```json
{
  "lockfile": ".trove.lock.yaml",
  "packages": [
    {
      "package": "nwks/platform/agent-backend",
      "installedVersion": "1.0.0",
      "installedDigest": "sha256:abc123...",
      "latestVersion": "1.0.0",
      "status": "current"
    },
    {
      "package": "nwks/frontend/react-defaults",
      "installedVersion": "2.0.0",
      "installedDigest": "sha256:def456...",
      "latestVersion": "2.1.0",
      "status": "update-available"
    }
  ]
}
```

### Status Values

| Status | Meaning |
|---|---|
| `current` | Installed version matches latest |
| `update-available` | A newer version exists |
| `yanked` | Installed version has been yanked |
| `deprecated` | Installed version is deprecated |

### Adoption Reporting

When configured with an API token, `trove check` can report adoption:

```bash
trove check --lock .trove.lock.yaml --report
```

### Error Cases

| Error | Cause |
|---|---|
| Lockfile not found | The specified lockfile doesn't exist |
| `PACKAGE_NOT_FOUND` | A referenced package no longer exists |
| `UNAUTHORIZED` | Authentication required for a private package |
| Server unreachable | The Trove server is not accessible |

### Next Steps

- [update](/cli/update) — apply available updates safely
- [Lockfiles](/cli/lockfiles) — understand the lockfile format
