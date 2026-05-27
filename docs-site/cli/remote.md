# trove remote

## Why

Use `trove remote` to store where an editable worktree publishes. Remotes live in the `local:` section of `Trovefile` and are local to the checkout.

## How

### Add A Remote

```bash
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
```

Short package refs are accepted when `TROVE_SERVER_URL` is set:

```bash
export TROVE_SERVER_URL=https://trove.company.com
trove remote add origin nwks/platform/agent-defaults
```

### List Remotes

```bash
trove remote list
trove remote list --json
```

### Replace Or Remove

```bash
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults --force
trove remote remove origin
```

## Reference

| Form | Description |
|---|---|
| `trove remote add <name> <url-or-package-ref>` | Add a remote without a network call |
| `trove remote list` | Print remotes in stable name order |
| `trove remote remove <name>` | Remove a configured remote |

Remote URLs use `https://trove.company.com/{org}/{namespace}/{package}`. Remote package refs must not include selectors.

## Next

- [status](/cli/status) checks whether the worktree can publish
- [push](/cli/push) publishes to the configured remote
