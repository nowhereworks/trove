# trove download

## Why

Use `trove download` when you need one artifact file without installing a package or writing Trove metadata. This is the fastest path for copying a shared `AGENTS.md` into a repo for immediate use.

`trove fetch` is not supported. Use `trove download` for single-artifact retrieval.

## How

### Download To A File

```bash
trove download nwks/platform/agent-backend AGENTS.md --output AGENTS.md
```

The package selector is optional. When omitted, Trove resolves `stable`.

### Stream To Stdout

```bash
trove download nwks/platform/agent-backend@1.0.0 AGENTS.md > AGENTS.md
```

When streaming bytes to stdout, the command does not print status text.

### Download A Bundled Core Skill

Trove ships some built-in agent skills outside package storage. Download them with their core skill path:

```bash
trove download core/skills/find-trove-skills/SKILL.md --output SKILL.md
```

Core skill downloads use the same `--output`, `--overwrite`, and `--json` flags as package artifact downloads.

### Overwrite Existing Files

```bash
trove download nwks/platform/agent-backend AGENTS.md --output AGENTS.md --overwrite
```

Existing different files are not overwritten unless `--overwrite` is set. Existing identical files succeed unchanged.

### JSON Metadata

```bash
trove download nwks/platform/agent-backend AGENTS.md --output AGENTS.md --json
```

Output:

```json
{
  "artifactPath": "AGENTS.md",
  "changed": "true",
  "output": "AGENTS.md",
  "package": "nwks/platform/agent-backend",
  "packageDigest": "sha256:abc123...",
  "selector": "stable",
  "version": "1.0.0"
}
```

Use `--metadata-only` to resolve without writing artifact bytes:

```bash
trove download nwks/platform/agent-backend AGENTS.md --metadata-only --json
```

## Reference

| Flag | Required | Description |
|---|---|---|
| `--output <file>` | No | Write artifact bytes to a file instead of stdout |
| `--overwrite` | No | Replace an existing different output file |
| `--json` | No | Emit metadata as JSON; requires `--output` or `--metadata-only` |
| `--metadata-only` | No | Resolve and report metadata without downloading bytes |

| Form | Description |
|---|---|
| `trove download <org/namespace/package[@selector]> <artifact-path>` | Download a package artifact |
| `trove download core/skills/<name>/SKILL.md` | Download a bundled core skill |

`trove download` does not write `.trove.lock.yaml`, `trove.yaml`, or `.trove/config.yaml`.

## Next

- [install](/cli/install) installs required artifacts and writes a lockfile
- [resolve](/cli/resolve) shows the exact version behind a selector
