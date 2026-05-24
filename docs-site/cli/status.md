# trove status

## Why

Use `trove status` before pushing an editable `AGENTS.md` package. It validates local files, reports the configured remote, and shows the next version Trove will use.

## How

```bash
trove status
```

Example output:

```text
Artifact: AGENTS.md
Remote: origin -> https://trove.company.com/nwks/platform/agent-defaults
Current published version: 1.0.0
Next version: 1.0.1
Visibility: private
Channel: stable
Review policy: requiresApproval
Local state: ready
```

For agents and CI:

```bash
trove status --json
```

## Reference

`trove status` checks for `AGENTS.md`, `trove.yaml`, `.trove/config.yaml`, a configured remote, manifest shape, package mismatch, and `TROVE_TOKEN` for publishing.

It exits non-zero when local state is blocked.

## Next

- [push](/cli/push) uploads and publishes the package
- [remote](/cli/remote) fixes missing remote configuration
