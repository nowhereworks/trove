# trove init

## Why

Use `trove init agents-md` to turn a repo-root `AGENTS.md` into an editable Trove package worktree without hand-writing Trove metadata.

## How

### Start A New AGENTS.md Package

```bash
trove init agents-md \
  --remote https://trove.company.com/nwks/platform/agent-defaults \
  --maintainer-team platform-engineering \
  --yes
```

The command creates or adopts `AGENTS.md`, writes `trove.yaml`, and writes `.trove/config.yaml`.

### Convenience Form

```bash
trove init --agents-md
```

### Package Before Remote

```bash
trove init agents-md --package nwks/platform/agent-defaults
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
```

Use this when you know the package coordinate before the server URL.

## Reference

| Flag | Required | Description |
|---|---|---|
| `--remote <url-or-package-ref>` | No | Configure the publishing remote during init |
| `--package <org/namespace/package>` | No | Set package coordinates without requiring a remote |
| `--display-name <name>` | No | Set `metadata.displayName` |
| `--description <text>` | No | Set `metadata.description` |
| `--visibility private\|internal\|public` | No | Set default package visibility |
| `--maintainer-team <team>` | No | Add a team maintainer |
| `--maintainer-user <user>` | No | Add a user maintainer |
| `--force` | No | Rewrite generated metadata files; never overwrites `AGENTS.md` |
| `--json` | No | Emit machine-readable output |

`trove init agents-md` does not call the network.

## Next

- [remote](/cli/remote) manages publishing remotes
- [push](/cli/push) uploads and publishes the package
