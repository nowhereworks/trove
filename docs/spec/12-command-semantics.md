# Command Semantics

## Goal

Keep Trove references, raw URLs, and CLI commands consistent across artifact types. These rules apply to `AGENTS.md`, skills, prompts, commands, templates, context packs, and future agent-consumable packages.

The CLI should feel GitHub/npm-like without making users understand Trove's internal draft, manifest, upload, review, and publish APIs.

## Package References

The canonical package identity is:

```text
org/namespace/package
```

A package selector appends `@selector`:

```text
org/namespace/package@selector
```

If a consumer command omits the selector, the CLI resolves `stable` by default:

```bash
trove download nwks/platform/agent-defaults AGENTS.md
trove install nwks/platform/agent-defaults --target .
```

Publishing commands must resolve omitted versions through their own version-selection rules. They must not silently overwrite a published version.

## Selectors

Supported selectors:

| Selector | Meaning |
|---|---|
| omitted | Same as `stable` for consumer reads |
| `stable` | Highest SemVer published, non-yanked version in the `stable` channel |
| `latest` | Highest SemVer published, non-yanked version regardless of channel |
| `1.2.3` | Exact immutable published version |
| `v1.2.3` | Ergonomic exact version, normalized to `1.2.3` |
| `v1` | Highest published, non-yanked `1.x.x` version |
| `v1.2` | Highest published, non-yanked `1.2.x` version |
| `sha256:<digest>` | Exact package-version digest selector |

MVP named aliases are only `latest` and `stable`. Major and minor selectors are derived from SemVer and are not user-managed channels.

`stable` is assigned by publishing a version with channel `stable`:

```yaml
spec:
  version: 1.0.3
  channel: stable
```

or with a CLI override:

```bash
trove push --channel stable
```

The server derives `stable` by querying published versions where `channel = stable`, excluding yanked versions, ordered by SemVer descending.

## Raw Artifact URLs

Raw artifact URLs use GitHub Actions-style trailing selectors:

```text
/raw/{org}/{namespace}/{package}/{artifactPath...}[@selector]
```

Examples:

```text
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@latest
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@v1.0.3
```

If `@selector` is omitted, the server resolves `stable`.

Alias or omitted-selector URLs must redirect to exact immutable URLs:

```text
GET /raw/nwks/platform/agent-defaults/AGENTS.md
302 /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3

GET /raw/nwks/platform/agent-defaults/AGENTS.md@stable
302 /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3

GET /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3
200 artifact bytes
```

Alias responses use `Cache-Control: no-cache`. Exact version responses use immutable cache headers and include the artifact digest as `ETag`.

Artifact paths must not contain `@`. Keeping `@` reserved for selectors avoids ambiguous raw URLs and matches GitHub Actions reference ergonomics.

Raw URLs require authentication by default. Public packages may allow anonymous reads when visibility and server configuration permit it. Private and internal raw reads can still be used with curl by passing a bearer token:

```bash
curl -L -H "Authorization: Bearer $TROVE_TOKEN" \
  https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
```

## Command Families

Trove CLI commands fall into these families.

### Resolve

`trove resolve` turns a package reference into an exact version and digest.

```bash
trove resolve nwks/platform/agent-defaults@stable
trove resolve nwks/platform/agent-defaults
```

Output must include the requested selector, resolved version, package digest, manifest URL, and archive URL. `--json` emits the same data as structured JSON.

### Download

`trove download` retrieves one artifact for immediate use.

```bash
trove download nwks/platform/agent-defaults AGENTS.md
trove download nwks/platform/agent-defaults@v1.0.3 AGENTS.md --output AGENTS.md
```

Behavior:

- Resolve omitted selectors to `stable`.
- Fetch exactly one artifact path.
- Write to stdout by default when no output path is provided.
- Write to `--output <file>` when provided.
- Never write lockfiles or project metadata.
- Support `--json` for metadata when not streaming bytes to stdout.
- `--json --output <file>` writes artifact bytes to the file and metadata to stdout.
- `--json --metadata-only` writes metadata to stdout and does not fetch or write artifact bytes.

`trove fetch` is not supported. `trove download` is the only single-artifact retrieval command.

### Install

`trove install` applies a package to a consuming project.

```bash
trove install nwks/platform/agent-defaults --target .
```

Behavior:

- Resolve omitted selectors to `stable`.
- Read the manifest.
- Install required artifacts by default.
- Install optional artifacts only when explicitly selected.
- Use each artifact's `targetPath`, defaulting to `path` when omitted.
- Refuse to overwrite existing different files unless overwrite is explicit.
- Write or update `.trove.lock.yaml` with exact versions and digests.
- Optionally report adoption when configured with an API token.

### Worktree

Worktree commands create and maintain local editable package checkouts for maintainers.

```bash
trove clone nwks/platform/agent-defaults@stable
trove pull
trove status
trove push --patch
```

`trove clone` retrieves the package manifest as local `trove.yaml`, downloads all selected artifacts, generates local Trove project config, and writes local baseline state for clean-change detection. It prepares a directory that can later run `trove status` and `trove push`.

`trove pull` refreshes the local editable checkout from the configured remote. It must not overwrite local changes silently. If remote changes conflict with local files, it fails with a clear conflict report. If local baseline state is missing or incomplete, it refuses automatic overwrite.

`trove status` compares the local checkout with its configured remote and reports missing auth, missing files, manifest problems, current published version, selected next version, and review policy.

`trove push` creates or updates a draft version, uploads the manifest first, uploads artifact files, then publishes or submits for review.

Published versions are immutable. Editing a cloned published version always prepares a new draft version; it does not mutate the source version.

### Remote

`trove remote` manages where a local editable worktree publishes.

```bash
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
trove remote add origin nwks/platform/agent-defaults
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults --force
trove remote list
trove remote remove origin
```

Accepted remote forms:

```text
https://trove.company.com/{org}/{namespace}/{package}
{org}/{namespace}/{package}
```

If only a package ref is provided, resolve the server from `TROVE_SERVER_URL` first, then existing `.trove/config.yaml`. If no server is known, fail with a clear error asking for a full URL.

`trove remote add` does not require a network call. It fails when the remote already exists unless `--force` is provided. `trove status`, `trove pull`, and `trove push` validate the remote when they need server state.

## Status

`trove status` shows what will happen before publishing from an editable worktree.

```text
Remote: origin -> https://trove.company.com/nwks/platform/agent-defaults
Current published version: 1.0.0
Next version: 1.0.1
Visibility: private
Channel: stable
Review policy: requires approval
Local state: ready to push
```

Status must report missing token, missing remote, missing manifest, missing artifact files, manifest/package mismatch, invalid version selection, and review blockers clearly.

## Push

`trove push` runs the publishing workflow for an editable worktree.

Default behavior:

- Resolve the configured remote package.
- Create the package if it does not exist and the user is authorized.
- Pick the next patch version automatically.
- Generate or update allowed manifest fields for that version.
- Create or reuse a draft version.
- Upload `trove.yaml` first.
- Upload manifest-listed artifacts.
- Try to publish.
- If publishing returns `APPROVAL_REQUIRED`, submit for review and print the review URL instead of failing.

Common flags:

| Flag | Description |
|---|---|
| `--patch` | Select next patch version; default |
| `--minor` | Select next minor version |
| `--major` | Select next major version |
| `--version 1.2.3` | Use an explicit strict SemVer version |
| `--remote origin` | Select a configured remote |
| `--visibility private\|internal\|public` | Override visibility for this push |
| `--channel stable` | Publish with the stable channel |
| `--draft` | Upload draft only; do not submit or publish |
| `--submit-only` | Upload and submit for review |
| `--publish` | Require publish to succeed; fail if review blocks it |
| `--yes` | Disable prompts |
| `--json` | Emit machine-readable output |

## Version Selection

Default publishing version selection should stay invisible unless the user needs control.

Rules:

- No published versions: `1.0.0`.
- Existing latest `1.0.0`: next default is `1.0.1`.
- `--minor`: increment minor and reset patch to zero.
- `--major`: increment major and reset minor and patch to zero.
- `--version`: use the exact user-provided strict SemVer version.
- If the selected version already exists as draft, reuse and update it.
- If the selected version already exists as published, fail and suggest the next patch version.

Published versions must use strict SemVer `MAJOR.MINOR.PATCH`. Prerelease and arbitrary string versions are deferred.

## Local Files

Consumer installs use `.trove.lock.yaml`. This file pins exact versions and digests for downstream projects.

Editable worktrees use `trove.yaml`, `.trove/config.yaml`, and optional `.trove/state.yaml`. These files describe the package, remember the publishing remote, and track the clean checkout baseline.

Do not use `.trove.lock.yaml` as the publishing source of truth.

`.trove/config.yaml` is local CLI state, not the registry package manifest.

```yaml
apiVersion: trove.io/v1
kind: TroveProject
defaultRemote: origin
artifactKind: agents-md
remotes:
  origin:
    serverUrl: https://trove.company.com
    package: nwks/platform/agent-defaults
publish:
  channel: stable
  visibility: private
```

Package-only initialization may omit `defaultRemote` and `remotes`; commands that require server state must then fail clearly until a remote is configured.

`.trove/state.yaml` is local baseline state written by clone/pull-capable workflows. It records the source remote, requested selector, resolved version, package digest, and per-file digests used to avoid overwriting local changes silently.

`trove.yaml` remains the actual package manifest uploaded to Trove.

`trove push` may update generated manifest fields, especially package coordinates, `spec.version`, `spec.channel`, `spec.visibility`, and `spec.lifecycle`. It must not rewrite artifact content files. Local editable manifests remain draft-oriented; published lifecycle is represented by server version state and publish responses.

## API Requirements

Write-side CLI clients need methods for package lookup, package creation, version lookup, draft creation, artifact upload, publishing, review submission, and approval status.

Draft reuse requires a version lookup that distinguishes reusable draft versions from immutable published, deprecated, or yanked versions. Review fallback requires publish failures blocked by approval to use the structured `APPROVAL_REQUIRED` error code.

Initially continue using `TROVE_SERVER_URL` and `TROVE_TOKEN`. A polished `trove login` command can come later.

Review-gated publishing requires routes that accept human-readable package coordinates and SemVer versions to resolve the underlying package version ID before calling review services.

## Output Rules

- Commands default to human-readable text.
- Commands that return structured metadata must support `--json`.
- Commands that stream artifact bytes to stdout must not mix progress text with the byte stream.
- Non-interactive commands fail with clear errors instead of prompting when required inputs are missing.
- Errors should include the package reference, selector, resolved version when known, and server request ID when available.
