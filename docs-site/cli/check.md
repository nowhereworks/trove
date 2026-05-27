# trove check

## Why

Your project has installed packages pinned in `.trove.lock.yaml`. Are those versions still current? Has a version been yanked? Is there a newer release? `trove check` answers these questions without changing anything — it's advisory by default, perfect for CI.

## How

### Basic Usage

```bash
trove check
```

Output:

```
nwks/platform/agent-backend: up to date
nwks/frontend/react-defaults: update available: 2.1.0
```

### Exit Codes

| Exit Code | Meaning |
|---|---|
| `0` | Command completed, including advisory update results |
| non-zero | Error reading the lockfile or contacting the server |

### CI Integration

```bash
# Advisory mode (default) — exits 0 even if updates are available
trove check
```

### JSON Output

```bash
trove check --json
```

Output:

```json
{
  "results": [
    {
      "package": "nwks/platform/agent-backend",
      "currentVersion": "1.0.0",
      "latestVersion": "1.0.0",
      "updateAvailable": false
    },
    {
      "package": "nwks/frontend/react-defaults",
      "currentVersion": "2.0.0",
      "latestVersion": "2.1.0",
      "updateAvailable": true
    }
  ],
  "hasUpdateAvailable": true
}
```

### Status Values

Human output reports either `up to date` or `update available: <version>` for each install.

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
