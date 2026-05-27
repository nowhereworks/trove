# trove pull

## Why

Use `trove pull` inside a cloned publishing worktree to refresh local files from the remote package without overwriting local edits silently.

## How

```bash
trove clone nwks/platform/agent-defaults@latest
cd agent-defaults
trove pull
```

The command reads `Trovefile` and `.trove/state.yaml`, resolves the original selector, downloads the remote manifest, merges it with local config, and updates `.trove/state.yaml`.

### Conflict Protection

`trove pull` only overwrites files that still match the baseline recorded in `.trove/state.yaml`. If `AGENTS.md` or `Trovefile` has local changes, the command stops and reports the conflicting path.

### JSON Output

```bash
trove pull --json
```

Output:

```json
{
  "package": "nwks/platform/agent-defaults",
  "version": "1.0.1"
}
```

## Reference

| Flag | Required | Description |
|---|---|---|
| `--remote <name>` | No | Select a configured remote when state does not pin one |
| `--json` | No | Emit machine-readable output |

`trove pull` refuses to overwrite files when `.trove/state.yaml` is missing or incomplete.

## Next

- [status](/cli/status) checks publishing readiness
- [push](/cli/push) uploads and publishes your changes
