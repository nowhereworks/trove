# trove update

## Why

You've run `trove check` and found updates available. Now you want to apply them — but safely. `trove update` shows you what will change before it changes anything. It's dry-run by default, so you can review diffs before writing files.

## How

### Dry-Run (Default)

```bash
trove update
```

Output:

```
Dry run. Use --apply to update lockfile and artifacts.
nwks/frontend/react-defaults: update available 2.0.0 -> 2.1.0
```

### Applying Updates

```bash
trove update --apply
```

Output:

```
nwks/frontend/react-defaults: updated 2.0.0 -> 2.1.0
Lockfile updated: /path/to/project/.trove.lock.yaml
```

### What Happens During Apply

1. Checks each installed package for a newer latest version
2. Downloads required artifacts for updated packages
3. Writes required artifacts to their target paths
4. Rewrites `.trove.lock.yaml` with new versions and digests

### JSON Output

```bash
trove update --json
```

Output:

```json
{
  "results": [
    {
      "package": "nwks/frontend/react-defaults",
      "currentVersion": "2.0.0",
      "latestVersion": "2.1.0",
      "updated": false
    }
  ],
  "applied": false
}
```

### Error Cases

| Error | Cause |
|---|---|
| Lockfile not found | The specified lockfile doesn't exist |
| `UNAUTHORIZED` | Authentication required |

### Next Steps

- [check](/cli/check) — verify your installed versions before updating
- [Lockfiles](/cli/lockfiles) — understand the lockfile format
