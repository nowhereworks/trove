# trove update

## Why

You've run `trove check` and found updates available. Now you want to apply them — but safely. `trove update` shows you what will change before it changes anything. It's dry-run by default, so you can review diffs and compatibility before writing files.

## How

### Dry-Run (Default)

```bash
trove update --lock .trove.lock.yaml
```

Output:

```
Checking for updates...

nwks/frontend/react-defaults
  Current: 2.0.0 (sha256:def456...)
  Latest:  2.1.0 (sha256:ghi789...)
  Compatibility: compatible

  Changed artifacts:
    ~ AGENTS.md (3840 → 4096 bytes)
    ~ styles/tailwind-preset.js (512 → 640 bytes)
    + context/react-19-migration.md (new, 1024 bytes)

  Lockfile changes:
    - version: 2.0.0 → 2.1.0
    - digest: sha256:def456... → sha256:ghi789...

Run with --apply to write changes.
```

### Applying Updates

```bash
trove update --lock .trove.lock.yaml --apply
```

Output:

```
Applying updates...

nwks/frontend/react-defaults
  Updating 2.0.0 → 2.1.0
  ✓ AGENTS.md (updated)
  ✓ styles/tailwind-preset.js (updated)
  ✓ context/react-19-migration.md (installed)
  ✓ .trove.lock.yaml (rewritten)

Updates applied. 1 package updated.
```

### What Happens During Apply

1. Resolves each package's latest compatible version
2. Downloads changed artifacts
3. Writes artifacts to their target paths (fails if target differs and `--overwrite` is not set)
4. Rewrites `.trove.lock.yaml` with new versions and digests
5. Reports adoption (if configured with a token)

### Overwriting Conflicting Files

If a target file has local modifications:

```bash
trove update --lock .trove.lock.yaml --apply
# ERROR: AGENTS.md has local modifications. Use --overwrite to replace.
```

To force overwrite:

```bash
trove update --lock .trove.lock.yaml --apply --overwrite
```

### JSON Output

```bash
trove update --lock .trove.lock.yaml --json
```

Output:

```json
{
  "updates": [
    {
      "package": "nwks/frontend/react-defaults",
      "fromVersion": "2.0.0",
      "toVersion": "2.1.0",
      "fromDigest": "sha256:def456...",
      "toDigest": "sha256:ghi789...",
      "artifactsChanged": [
        { "path": "AGENTS.md", "action": "updated", "fromBytes": 3840, "toBytes": 4096 },
        { "path": "styles/tailwind-preset.js", "action": "updated", "fromBytes": 512, "toBytes": 640 },
        { "path": "context/react-19-migration.md", "action": "added", "toBytes": 1024 }
      ]
    }
  ],
  "lockfileUpdated": true
}
```

### Compatibility Checks

`trove update` checks compatibility before applying:

```bash
trove update --lock .trove.lock.yaml
# nwks/ai/security-reviewer
#   Latest: 1.0.0 → 2.0.0
#   Compatibility: incompatible (requires opencode >=1.0.0, you have 0.6.0)
#   Skipping incompatible update.
```

Incompatible updates are skipped by default. Use `--include-incompatible` to force them (not recommended).

### Adoption Reporting

```bash
trove update --lock .trove.lock.yaml --apply --report
```

### Error Cases

| Error | Cause |
|---|---|
| Lockfile not found | The specified lockfile doesn't exist |
| File conflict | Target file has local changes (use `--overwrite`) |
| `UNAUTHORIZED` | Authentication required |
| Incompatible | Latest version doesn't match your constraints |

### Next Steps

- [check](/cli/check) — verify your installed versions before updating
- [Lockfiles](/cli/lockfiles) — understand the lockfile format
