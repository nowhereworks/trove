# trove clone

## Why

Use `trove clone` when you maintain a published package and want an editable worktree for the next immutable version.

## How

```bash
trove clone nwks/platform/agent-defaults@stable
cd agent-defaults
trove status
trove push --patch
```

The clone writes:

- `trove.yaml`
- `AGENTS.md`
- `.trove/config.yaml`
- `.trove/state.yaml`

It does not write `.trove.lock.yaml` because this is a publishing worktree, not a consuming repo install.

### Choose The Directory

```bash
trove clone nwks/platform/agent-defaults@stable agent-defaults-edit
```

The target directory must not already contain files.

## Reference

| Command | Description |
|---|---|
| `trove clone <package-ref>` | Clone into a directory named after the package |
| `trove clone <package-ref> <dir>` | Clone into a chosen directory |

Omitted selectors resolve to `stable`.

## Next

- [push](/cli/push) publishes the next version
- [remote](/cli/remote) shows the configured publishing remote
