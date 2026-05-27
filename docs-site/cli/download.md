# trove download

## Why

Use `trove download` when you need package files without installing a package or writing Trove metadata. This is the fastest path for copying shared agent artifacts into a repo for immediate use.

`trove fetch` is not supported. Use `trove download` for single-artifact retrieval.

## How

### Download A Package

```bash
trove download http://localhost:8080/nwks/platform/agent-backend@latest --output .
```

The full URL form is useful before a repo has any Trove config. It uses the URL host as the Trove server and downloads the package artifacts into the output directory.

`trove download` writes only package artifact files. It does not write `Trovefile`, `.trove/*`, or `.trove.lock.yaml`.

### Download One Artifact To A File

```bash
trove download nwks/platform/agent-backend AGENTS.md --output AGENTS.md
```

The package selector is optional. When omitted, Trove resolves `latest`.

### Stream To Stdout

```bash
trove download nwks/platform/agent-backend@1.0.0 AGENTS.md > AGENTS.md
```

When streaming one artifact's bytes to stdout, the command does not print status text.

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
  "selector": "latest",
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
| `trove download <org/namespace/package[@selector]>` | Download package artifact files |
| `trove download <server-url>/<org>/<namespace>/<package>[@selector]` | Download package artifact files from a server without prior config |
| `trove download <org/namespace/package[@selector]> <artifact-path>` | Download one package artifact |
| `trove download core/skills/<name>/SKILL.md` | Download a bundled core skill |

For package downloads, `--output` is a directory. For one-artifact and core skill downloads, `--output` is a file path.

## Next

- [install](/cli/install) installs required artifacts and writes a lockfile
- [resolve](/cli/resolve) shows the exact version behind a selector
