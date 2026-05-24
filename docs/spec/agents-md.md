# AGENTS.md CLI Publishing Implementation Spec

## Status

Implemented.

This document defines the `AGENTS.md` specialization of the Trove command model. The general command, package reference, selector, raw URL, versioning, and worktree rules live in [`12-command-semantics.md`](12-command-semantics.md). If this document and `12-command-semantics.md` conflict, update both before implementing.

## Goal

Make publishing a repo-root `AGENTS.md` to Trove feel natural, ergonomic, and low-bureaucracy for developers, platform teams, and citizen developers.

The common path must hide Trove's internal draft, manifest, upload, review, and publish workflow behind a Git-like flow:

```bash
trove init agents-md
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
trove status
trove push
```

The equivalent convenience form is also supported:

```bash
trove init --agents-md
```

Do not implement `trove agent init`. That name is reserved for future subagent or `agents/` artifact workflows.

## Scope

This feature covers publishing exactly one repo-root instruction file as a Trove package artifact:

```text
AGENTS.md
```

`trove.yaml` is required package manifest transport. It is uploaded before `AGENTS.md` through the artifact upload endpoint so the server can validate and type later uploads, but it is not a manifest-listed installable artifact in the `agents-md` workflow.

The generated package manifest must declare that file as:

```yaml
artifacts:
  - path: AGENTS.md
    type: agent-instructions
    required: true
    targetPath: AGENTS.md
```

The generated package manifest must not list `trove.yaml` under `spec.artifacts` for this workflow.

The `agents-md` specialization may later coexist with package types that include skills, subagents, commands, prompts, templates, and context packs. This spec does not implement those package authoring flows.

## Non-Goals

- Do not add browser-based editing for `AGENTS.md`.
- Do not add Git-backed package storage.
- Do not add `trove login`; continue using `TROVE_SERVER_URL` and `TROVE_TOKEN` for initial CLI auth.
- Do not implement future `trove agent` or `trove agents` command families.
- Do not support prerelease versions or arbitrary version strings.
- Do not preserve comments or hand formatting in generated `trove.yaml` or `.trove/config.yaml` when the CLI rewrites them.
- Do not support `trove fetch`. `trove download` is the only single-artifact retrieval command.

## Retrieval Contract

A published `AGENTS.md` can be retrieved in four ways:

| Use case | Command or URL | Writes Trove metadata |
|---|---|---|
| Raw HTTP read | `/raw/{org}/{namespace}/{package}/AGENTS.md[@selector]` | No |
| One-file download | `trove download <package-ref> AGENTS.md` | No |
| Consumer install | `trove install <package-ref> --target .` | Yes, `.trove.lock.yaml` |
| Maintainer checkout | `trove clone <package-ref>` | Yes, `trove.yaml`, `.trove/config.yaml`, and `.trove/state.yaml` |

### Raw URL

Raw artifact URLs use trailing selectors:

```text
/raw/{org}/{namespace}/{package}/AGENTS.md[@selector]
```

Examples:

```text
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@latest
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@v1.0.3
```

If the selector is omitted, the server resolves `stable`.

Alias or omitted-selector URLs redirect to exact immutable URLs:

```text
GET /raw/nwks/platform/agent-defaults/AGENTS.md
302 /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3

GET /raw/nwks/platform/agent-defaults/AGENTS.md@stable
302 /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3

GET /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3
200 AGENTS.md bytes
```

Exact raw responses include the artifact digest as `ETag`. Alias responses use `Cache-Control: no-cache`. Exact public responses use immutable public cache headers. Exact private responses use immutable private cache headers.

Raw URLs require authentication by default. Public packages may allow anonymous reads when effective visibility and server configuration permit it.

Private and internal packages can be fetched with curl by passing a bearer token:

```bash
curl -L -H "Authorization: Bearer $TROVE_TOKEN" \
  https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
```

Artifact paths must not contain `@`; raw artifact URLs reserve `@` for selectors.

### Download For Immediate Use

Use `trove download` when the caller wants one artifact file and does not want Trove project metadata.

```bash
trove download nwks/platform/agent-defaults AGENTS.md --output AGENTS.md
trove download nwks/platform/agent-defaults@v1.0.3 AGENTS.md --output AGENTS.md
trove download nwks/platform/backend-skills skills/reviewer/SKILL.md --output .opencode/skills/reviewer/SKILL.md
```

Behavior:

- Omitted selector resolves to `stable`.
- Exactly one artifact path is fetched.
- The artifact path is the manifest `path`, not `targetPath`.
- No `.trove.lock.yaml`, `trove.yaml`, or `.trove/config.yaml` is written.
- No archive is downloaded or extracted.
- No progress or status text is printed to stdout when streaming bytes.
- Existing files are not overwritten unless `--overwrite` is provided.
- Parent directories for `--output` are created when needed.
- `trove fetch` is not accepted as an alias.

### Install Into A Consuming Repo

Use `trove install` when a downstream project adopts the package and wants update tracking.

```bash
trove install nwks/platform/agent-defaults --target .
```

Behavior:

- Resolve the package to an exact published version.
- Read the manifest.
- Install required artifacts, including `AGENTS.md`.
- Use `targetPath`, defaulting to `path` when omitted.
- Refuse to overwrite different existing files unless overwrite is explicit.
- Write or update `.trove.lock.yaml` with exact version and digest pins.
- Do not write `trove.yaml` or `.trove/config.yaml`.

### Clone For Maintainer Edits

Use `trove clone` when a maintainer wants to edit a published package and later publish a new immutable version.

```bash
trove clone nwks/platform/agent-defaults@stable
cd agent-defaults
trove status
trove push --patch
```

Behavior:

- Resolve the selected published version.
- Retrieve the version manifest as `trove.yaml` and download all manifest-listed artifacts needed for the editable checkout.
- Write local `.trove/config.yaml` with the remote and default publishing settings.
- Prepare future `trove status`, `trove pull`, and `trove push` commands.
- Treat edits as changes for a new draft version.
- Never mutate the cloned published version.

`trove pull` refreshes the editable checkout from the configured remote and must not overwrite local changes silently.

## Command Contracts

Commands default to human-readable text. Commands that return structured metadata support `--json`. Commands that stream artifact bytes to stdout must not mix human-readable text into stdout.

### `trove init agents-md`

Creates or adopts a repo-root `AGENTS.md` and sets up local Trove publishing metadata.

Supported forms:

```bash
trove init agents-md
trove init --agents-md
trove init agents-md --remote https://trove.company.com/nwks/platform/agent-defaults
trove init agents-md --package nwks/platform/agent-defaults
```

Flags:

| Flag | Description |
|---|---|
| `--remote <url-or-package-ref>` | Configure a publishing remote during init |
| `--package <org/namespace/package>` | Set the package coordinate without a server URL |
| `--display-name <name>` | Set manifest display name |
| `--description <text>` | Set manifest description |
| `--visibility private\|internal\|public` | Set default visibility |
| `--maintainer-team <team>` | Set a manifest team maintainer |
| `--maintainer-user <user>` | Set a manifest user maintainer |
| `--force` | Rewrite generated metadata files; never overwrite `AGENTS.md` |
| `--yes` | Disable prompts |
| `--json` | Emit machine-readable output |

Filesystem behavior:

- If `AGENTS.md` exists, reuse it unchanged.
- If `AGENTS.md` is missing, create a starter file.
- If `trove.yaml` is missing, create it.
- If `.trove/config.yaml` is missing, create it.
- If run again, be idempotent and non-destructive.
- If `--force` is used, generated metadata files may be rewritten.
- `--force` must never overwrite `AGENTS.md`.

Remote behavior:

- If `--remote` is a full URL, parse and store both server URL and package coordinate.
- If `--remote` is a package ref, resolve the server from `TROVE_SERVER_URL` or existing `.trove/config.yaml`.
- If `--package` is provided and a server is known from `TROVE_SERVER_URL` or existing `.trove/config.yaml`, create or update the default remote.
- If `--package` is provided and no server is known, write the package coordinate to `trove.yaml` only; do not create a remote with a missing `serverUrl`.
- `init` does not require a network call.

Default generated values:

| Field | Default |
|---|---|
| `metadata.displayName` | Title-cased package name |
| `metadata.description` | `Shared AGENTS.md instructions.` |
| `spec.version` | `1.0.0` |
| `spec.channel` | `stable` |
| `spec.visibility` | `private` |
| `spec.lifecycle` | `draft` |
| maintainer | Require flag, prompt, or leave invalid with clear status error in non-interactive mode |

Human output example:

```text
Initialized AGENTS.md package worktree
Artifact: AGENTS.md
Manifest: trove.yaml
Config: .trove/config.yaml
Remote: origin -> https://trove.company.com/nwks/platform/agent-defaults
```

JSON output shape:

```json
{
  "artifactKind": "agents-md",
  "artifactPath": "AGENTS.md",
  "manifestPath": "trove.yaml",
  "configPath": ".trove/config.yaml",
  "remote": "origin",
  "serverUrl": "https://trove.company.com",
  "package": "nwks/platform/agent-defaults"
}
```

### `trove remote`

Stores where the local editable worktree publishes.

Supported forms:

```bash
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
trove remote add origin nwks/platform/agent-defaults
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults --force
trove remote list
trove remote remove origin
```

Accepted remote values:

```text
https://trove.company.com/{org}/{namespace}/{package}
{org}/{namespace}/{package}
```

If only a package ref is provided, resolve the server from:

1. `TROVE_SERVER_URL`
2. Existing `.trove/config.yaml`
3. Otherwise return a clear error asking for a full URL

Behavior:

- Read and write `.trove/config.yaml`.
- Create `.trove/` if needed.
- `remote add` does not require a network call.
- `remote add` fails if the remote name already exists unless `--force` is provided.
- `remote remove` fails if the remote does not exist.
- `remote list --json` emits configured remotes in stable name order.

Flags:

| Flag | Description |
|---|---|
| `--force` | Replace an existing remote with the same name |
| `--json` | Emit machine-readable output for `list` |

If `--force` replaces the current `defaultRemote`, the default still points at the same remote name with the new value.

### Local State Without A Remote

`trove init agents-md --package nwks/platform/agent-defaults` can create a valid local manifest before the user knows the server URL. In that state:

- `trove.yaml` contains `metadata.org`, `metadata.namespace`, and `metadata.name` from `--package`.
- `.trove/config.yaml` may omit `defaultRemote` and `remotes`.
- `trove status` returns non-zero with a clear missing-remote problem.
- `trove push` returns non-zero and asks the user to run `trove remote add origin <url-or-package-ref>` or rerun init with `--remote`.

Example package-only config:

```yaml
apiVersion: trove.io/v1
kind: TroveProject
artifactKind: agents-md
publish:
  channel: stable
  visibility: private
```

Validation rules for `.trove/config.yaml` are in [Local Files](#local-files).

### `trove status`

Shows what will happen before publishing from an editable `AGENTS.md` worktree.

```text
Artifact: AGENTS.md
Remote: origin -> https://trove.company.com/nwks/platform/agent-defaults
Current published version: 1.0.0
Next version: 1.0.1
Visibility: private
Channel: stable
Review policy: requires approval
Local state: ready to push
```

Behavior:

- Load `.trove/config.yaml`.
- Load and validate `trove.yaml`.
- Verify `AGENTS.md` exists.
- Query remote package metadata when auth and remote configuration are available.
- Compute default next patch version.
- Report missing token, missing remote, missing manifest, missing `AGENTS.md`, manifest/package mismatch, invalid version selection, and review blockers clearly.
- Return non-zero when local state is not publishable.

JSON output shape:

```json
{
  "artifactKind": "agents-md",
  "artifactPath": "AGENTS.md",
  "remote": {
    "name": "origin",
    "serverUrl": "https://trove.company.com",
    "package": "nwks/platform/agent-defaults"
  },
  "currentPublishedVersion": "1.0.0",
  "nextVersion": "1.0.1",
  "visibility": "private",
  "channel": "stable",
  "reviewPolicy": "requiresApproval",
  "localState": "ready",
  "problems": []
}
```

### `trove push`

Runs the full publishing workflow for the local editable `AGENTS.md` worktree.

Default behavior:

- Load `.trove/config.yaml` and `trove.yaml`.
- Resolve the selected configured remote.
- Create the package if it does not exist and the caller is authorized.
- Pick the next patch version automatically.
- Generate or update allowed manifest fields for that version.
- Create or reuse a draft version.
- Upload `trove.yaml` first.
- Upload `AGENTS.md` second.
- Try to publish.
- If publishing returns `APPROVAL_REQUIRED`, submit for review and print the review URL instead of failing.

Flags:

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

Flag interactions:

- `--patch`, `--minor`, `--major`, and `--version` are mutually exclusive.
- `--draft`, `--submit-only`, and `--publish` are mutually exclusive.
- `--publish` returns non-zero if review approval blocks publication.
- Default mode attempts publish and falls back to submit-for-review when approval is required.
- Default mode falls back only for the structured `APPROVAL_REQUIRED` error code; other publish errors fail.
- `--submit-only` returns success when review submission succeeds.
- `--draft` returns success after uploads complete.

Required scopes:

| Mode | Required scopes |
|---|---|
| `--draft` | `package:write` |
| `--submit-only` | `package:write`, `review:write` |
| default | `package:write`, `version:publish`, and `review:write` when approval fallback is needed |
| `--publish` | `package:write`, `version:publish` |

Manifest rewrite rules:

- The CLI may rewrite generated fields in `trove.yaml`.
- The CLI must not rewrite `AGENTS.md`.
- The CLI must upload `trove.yaml` before `AGENTS.md` so artifact metadata can be typed from the manifest.
- Local editable `trove.yaml` remains draft-oriented. A successful publish changes server-side lifecycle to `published`, but the CLI does not rewrite local `spec.lifecycle` from `draft` to `published`.

Allowed generated manifest fields:

- `metadata.org`
- `metadata.namespace`
- `metadata.name`
- `metadata.displayName` when generated and not user-set
- `metadata.description` when generated and not user-set
- `spec.version`
- `spec.channel`
- `spec.visibility`
- `spec.lifecycle`
- `spec.artifacts` for the required `AGENTS.md` artifact

`spec.lifecycle` in the uploaded manifest should be `draft` for editable worktrees. The server publish response is the source of truth for published lifecycle state.

Human output when publish succeeds:

```text
Published nwks/platform/agent-defaults@1.0.0
Install with:
  trove install nwks/platform/agent-defaults@stable
```

Human output when review is required:

```text
Uploaded nwks/platform/agent-defaults@1.0.0
Submitted for review:
  https://trove.company.com/reviews/...
```

JSON output shape:

```json
{
  "package": "nwks/platform/agent-defaults",
  "version": "1.0.0",
  "lifecycle": "published",
  "digest": "sha256:abc123",
  "channel": "stable",
  "visibility": "private",
  "reviewUrl": "",
  "installCommand": "trove install nwks/platform/agent-defaults@stable"
}
```

### `trove download`

Downloads exactly one artifact from any package type. For this feature, the primary path is `AGENTS.md`.

```bash
trove download nwks/platform/agent-defaults AGENTS.md --output AGENTS.md
```

Flags:

| Flag | Description |
|---|---|
| `--output <file>` | Write artifact bytes to a file instead of stdout |
| `--overwrite` | Allow replacing an existing different output file |
| `--json` | Emit metadata instead of bytes; requires `--output` or `--metadata-only` |
| `--metadata-only` | Resolve and report metadata without writing artifact bytes |

Behavior:

- `trove download <package-ref> <artifact-path>` is required.
- The package ref may omit selector; omitted selector resolves `stable`.
- The command resolves the selector, then fetches the exact raw artifact.
- With no `--output`, bytes are written to stdout.
- With `--output`, bytes are written to the exact provided file path.
- Existing different files fail unless `--overwrite` is set.
- Existing identical files succeed and report unchanged when not streaming bytes.
- JSON mode must not emit artifact bytes to stdout.
- `--json --output <file>` writes artifact bytes to the file and metadata to stdout.
- `--json --metadata-only` writes metadata to stdout and does not fetch or write artifact bytes.
- `--json` without `--output` or `--metadata-only` fails because stdout is reserved for JSON.
- `trove fetch` must return an unknown subcommand error.

### `trove clone`

Creates an editable package worktree from an existing published version.

```bash
trove clone nwks/platform/agent-defaults@stable
trove clone nwks/platform/agent-defaults@stable agent-defaults
```

Behavior:

- Resolve omitted selector as `stable`.
- Default directory is the package name.
- Fail if target directory exists and is not empty.
- Retrieve the package version manifest and write it as local `trove.yaml`.
- Download manifest-listed artifact files such as `AGENTS.md`.
- Generate local `.trove/config.yaml` pointing at the package remote.
- Write `.trove/state.yaml` with the source selector, resolved version, package digest, manifest digest when known, and per-file digests for clean-change detection.
- Set `artifactKind: agents-md` when the manifest has exactly the required repo-root `AGENTS.md` artifact.
- Do not write `.trove.lock.yaml`.

### `trove pull`

Refreshes an editable package worktree from the configured remote.

Behavior:

- Load `.trove/config.yaml` and selected remote.
- Load `.trove/state.yaml` when present.
- Resolve configured selector or `stable` by default.
- Download remote manifest and artifacts.
- Compare remote files to local files before writing.
- Refuse to overwrite local changes silently.
- Report conflicts with file paths and suggested next actions.

If `.trove/state.yaml` is missing or incomplete, `trove pull` must refuse to overwrite existing files unless the user chooses an explicit future overwrite/rebase mode. This spec does not define that overwrite mode.

## Local Files

Editable publishing worktrees use `trove.yaml`, `.trove/config.yaml`, and optional `.trove/state.yaml`. Consumer installs use `.trove.lock.yaml`. Do not use `.trove.lock.yaml` as publishing source of truth.

### `.trove/config.yaml`

This is local CLI state, not the registry package manifest.

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

Validation rules:

- `apiVersion` must be `trove.io/v1`.
- `kind` must be `TroveProject`.
- `artifactKind` must be `agents-md` for this workflow.
- `defaultRemote` must exist in `remotes` when set.
- `defaultRemote` and `remotes` may be omitted only for package-only initialization that cannot yet publish.
- Remote package refs must be full `org/namespace/package` refs without selectors.
- Remote server URLs must be absolute `http` or `https` URLs.

### `.trove/state.yaml`

This is local pull/clone baseline state. It is not uploaded and is not used by consumer installs.

```yaml
apiVersion: trove.io/v1
kind: TroveProjectState
source:
  remote: origin
  requestedSelector: stable
  resolvedVersion: 1.0.0
  packageDigest: sha256:abc123
files:
  trove.yaml:
    digest: sha256:def456
  AGENTS.md:
    digest: sha256:789abc
```

Rules:

- `trove clone` writes `.trove/state.yaml` after writing checkout files.
- `trove pull` uses file digests from `.trove/state.yaml` to detect local changes before overwriting files.
- `trove push` may update `.trove/state.yaml` after a successful publish or draft upload, but server state remains authoritative.
- Missing or incomplete state is treated as unsafe for automatic overwrite.

### `trove.yaml`

This remains the package manifest uploaded to Trove.

```yaml
apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: nwks
  namespace: platform
  name: agent-defaults
  displayName: Agent Defaults
  description: Shared AGENTS.md instructions.
spec:
  version: 1.0.0
  channel: stable
  visibility: private
  lifecycle: draft
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
  maintainers:
    - team: platform-engineering
```

The manifest must pass the validation rules in [`02-manifest-and-lockfile.md`](02-manifest-and-lockfile.md).

For editable worktrees, `spec.lifecycle` stays `draft` before and after `trove push`. Published lifecycle is represented by server version state and publish responses, not by rewriting the local editable manifest to `published`.

For the `agents-md` workflow, `trove status` and `trove push` must additionally verify:

- `AGENTS.md` is present in the repo root.
- The manifest contains an artifact with `path: AGENTS.md`.
- That artifact has `type: agent-instructions`.
- That artifact has `required: true`.
- If `targetPath` is set, it is `AGENTS.md`.

## Version Selection

Default publishing version selection follows [`12-command-semantics.md`](12-command-semantics.md). For the standard `AGENTS.md` case, versioning should stay invisible unless the user needs control.

Rules:

- No published versions: `1.0.0`.
- Existing latest `1.0.0`: next default is `1.0.1`.
- `--minor`: increment minor and reset patch to zero.
- `--major`: increment major and reset minor and patch to zero.
- `--version`: use the exact user-provided strict SemVer version.
- If the selected version already exists as draft, reuse and update it.
- If the selected version already exists as published, fail and suggest the next patch version.

Published versions must use strict SemVer `MAJOR.MINOR.PATCH`. `v1.2.3` may be accepted as CLI input and normalized to `1.2.3`. Prerelease and arbitrary string versions are not supported.

## API Requirements

Initial CLI auth uses:

- `TROVE_SERVER_URL` for server selection when no remote server is configured.
- `TROVE_TOKEN` as a bearer token for authenticated APIs.

The CLI needs client methods for these APIs:

| Method | Endpoint | Purpose | Required scope |
|---|---|---|---|
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}` | Get package metadata and versions | visibility-dependent |
| `POST` | `/api/v1/packages` | Create package | `package:write` |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}` | Get one version, including draft lifecycle when authorized | visibility-dependent for published versions; maintainer for drafts |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions` | Create draft version | `package:write` |
| `PUT` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/trove.yaml` | Upload manifest | `package:write` |
| `PUT` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/AGENTS.md` | Upload `AGENTS.md` | `package:write` |
| `POST` | `/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/submit` | Submit for review | `review:write` |
| `GET` | `/api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/approval-status` | Check approval status | authenticated reviewer or maintainer |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish` | Publish version | `version:publish` |
| `GET` | `/api/v1/resolve/{org}/{namespace}/{package}@{selector}` | Resolve selector | visibility-dependent |
| `GET` | `/raw/{org}/{namespace}/{package}/{artifactPath...}@{selector}` | Download artifact | visibility-dependent |

Request bodies:

```json
POST /api/v1/packages
{
  "org": "nwks",
  "namespace": "platform",
  "name": "agent-defaults",
  "displayName": "Agent Defaults",
  "description": "Shared AGENTS.md instructions.",
  "visibility": "private"
}
```

```json
POST /api/v1/packages/nwks/platform/agent-defaults/versions
{
  "version": "1.0.0",
  "visibility": "private"
}
```

Version lookup response:

```json
GET /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.0
{
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-defaults",
  "version": "1.0.0",
  "lifecycle": "draft",
  "visibility": "private",
  "digest": "",
  "createdAt": "2026-05-24T00:00:00Z",
  "updatedAt": "2026-05-24T00:00:00Z",
  "publishedAt": ""
}
```

Create endpoints return `201 Created` with resource JSON. Upload, submit, approval status, and publish endpoints return `200 OK` with resource JSON or status JSON.

Draft reuse contract:

- `trove push` first checks `GET /api/v1/packages/{org}/{namespace}/{package}/versions/{version}` when selecting an explicit version or after a create conflict.
- If the existing version has `lifecycle: draft`, the CLI reuses it and uploads the new manifest/artifact bytes.
- If the existing version has `lifecycle: published`, `deprecated`, or `yanked`, the CLI fails and suggests the next patch version.
- If `POST /versions` returns `409 VERSION_ALREADY_EXISTS`, the error body must include `lifecycle` when the caller is authorized to see that version; otherwise the CLI performs the version lookup.

Version-exists error response:

```json
{
  "error": {
    "code": "VERSION_ALREADY_EXISTS",
    "message": "Version nwks/platform/agent-defaults@1.0.0 already exists.",
    "requestId": "req_123",
    "details": {
      "lifecycle": "draft"
    }
  }
}
```

Publish approval contract:

- When review approval blocks publication, `POST /publish` returns `403` with error code `APPROVAL_REQUIRED` and includes the request ID.
- Default `trove push` catches only `APPROVAL_REQUIRED`, then calls review submit.
- `trove push --publish` returns non-zero on `APPROVAL_REQUIRED` and does not submit for review.
- Other `403` errors, including missing `version:publish`, do not fall back to review submission.

API errors must include the common error shape and request ID. CLI errors should include package ref, selector, resolved version when known, and server request ID when available.

## Backend Requirements

### Trailing-Selector Raw URLs

Implement the canonical raw URL route:

```text
/raw/{org}/{namespace}/{package}/{artifactPath...}[@selector]
```

Required behavior:

- Omitted selector resolves `stable`.
- Alias selectors redirect to exact immutable trailing-selector URLs.
- Exact selectors return bytes.
- Exact `v1.2.3` selectors normalize to `1.2.3` in redirects and lookups.
- Artifact paths containing `@` are rejected.
- Private and internal reads require bearer auth unless dev auth is active.

The legacy selector-before-path raw route must not be used by new CLI code.

### Review Route Version Resolution

Review-gated publishing must resolve human-readable package coordinates and SemVer versions to package version UUIDs before calling review service methods.

Current implementation issue to fix:

- Review handlers pass the `{version}` route param, such as `1.0.0`, into review service methods that expect a package version UUID.
- Publish approval checks also call approval status with the SemVer string.
- Approval-gated publishing is therefore broken for real PostgreSQL-backed usage.

Implementation requirements:

- Add a query to resolve package version ID by `org`, `namespace`, `package`, and `version`.
- In review submit, list, approval status, and publish handlers, resolve the route SemVer to package version UUID before calling the review service.
- For approve and request-changes routes that are keyed by review ID, derive package version ID from the review row where possible instead of requiring clients to send `packageVersionId`.
- Enforce `review:write` on review mutation routes.
- Add tests for submit, approve, approval status, and publish with approval required.

## Implementation Slices

Implement in this order to minimize blocked CLI work.

### Slice 1: Raw URL Retrieval

Implement:

- Trailing-selector raw route `/raw/{org}/{namespace}/{package}/{artifactPath...}[@selector]`.
- Omitted selector resolves `stable`.
- Alias and omitted-selector URLs redirect to exact immutable URLs.
- Exact raw URLs return artifact bytes with digest `ETag`.
- Private and internal reads require bearer auth.
- Artifact paths containing `@` are rejected.

Tests:

- Exact version returns `200` and bytes.
- `stable` redirects to exact version.
- Omitted selector redirects to stable exact version.
- Private package without auth returns unauthorized or not found according to existing visibility policy.
- Artifact path with `@` returns `400`.

### Slice 2: Review Backend Fix

Implement:

- Package-version-ID lookup query.
- Review route SemVer-to-UUID resolution.
- Publish approval check using package version UUID.
- `review:write` enforcement on review mutation routes.
- Approval route no longer requires clients to send `packageVersionId` when the server can derive it from `reviewId`.

Tests:

- Submit review for `/versions/1.0.0/submit` creates review for the UUID-backed package version.
- Approval status for `/versions/1.0.0/approval-status` counts approvals.
- Publish fails before required approval.
- Publish succeeds after required approval.
- Missing `review:write` fails for review mutation routes.

### Slice 3: Local Project Config And Parsers

Implement:

- Read and write `.trove/config.yaml`.
- Full remote URL parser.
- Package ref parser with optional selector.
- Package ref parser without selector for remotes.
- Strict SemVer normalization for publishing inputs.
- Stable YAML output for generated config.
- Package-only init config with no remote.

Tests:

- Round-trip `.trove/config.yaml`.
- Full URL remote parsing.
- Short package ref remote parsing with `TROVE_SERVER_URL`.
- Invalid remote errors.
- Selector rejection for remote package refs.
- Package-only config validates but status/push report missing remote.

### Slice 4: Download Command

Implement:

- `trove download <package-ref> <artifact-path>`.
- `--output <file>`.
- `--overwrite`.
- `--json`.
- `--metadata-only`.
- Omitted package selector resolves `stable`.
- Existing files are not overwritten unless overwrite is explicit.
- No lockfile or publishing metadata writes.
- `trove fetch` is removed and returns unknown subcommand.

Tests:

- Streams artifact bytes to stdout by default.
- Writes to `--output`.
- Creates parent directories for output.
- Refuses to overwrite different file.
- Allows overwrite with `--overwrite`.
- JSON mode emits metadata and no bytes.
- `--json --output` writes bytes to file and metadata to stdout.
- `--json` without `--output` or `--metadata-only` fails.
- No `.trove.lock.yaml`, `trove.yaml`, or `.trove/config.yaml` is written.
- `trove fetch` fails as unknown subcommand.

### Slice 5: Clone And Pull

Implement:

- `trove clone <package-ref> [dir]`.
- `trove pull` from a cloned worktree.
- Retrieve the remote manifest as local `trove.yaml`, download `AGENTS.md`, and generate `.trove/config.yaml` plus `.trove/state.yaml`.
- Detect local changes before overwriting files.
- Prepare checkout for `trove status` and `trove push`.

Tests:

- Clone creates expected files.
- Clone writes `.trove/state.yaml` with source and file digests.
- Clone refuses non-empty directory.
- Pull updates clean files.
- Pull reports conflicts for locally changed files.
- Pull refuses automatic overwrite when `.trove/state.yaml` is missing.

### Slice 6: Init

Implement:

- Create or adopt `AGENTS.md`.
- Create or update `trove.yaml`.
- Create `.trove/config.yaml`.
- Support `trove init agents-md`.
- Support `trove init --agents-md`.
- Add CLI routing and tests.

Tests:

- Missing `AGENTS.md` creates starter file.
- Existing `AGENTS.md` is preserved.
- Re-running init is idempotent.
- `--remote` writes remote config.
- `--package` writes package coordinate.
- `--force` rewrites generated metadata but not `AGENTS.md`.

### Slice 7: Remote

Implement:

- `trove remote add`.
- `trove remote list`.
- `trove remote remove`.
- Full URL and short package ref support.
- No network call required for `remote add`.

Tests:

- Add full URL remote.
- Add short ref remote with server from env.
- Add with `--force` replaces an existing remote.
- List remotes in stable order.
- Remove remote.
- Missing server for short ref fails clearly.

### Slice 8: Status

Implement:

- Load local state.
- Query remote package if reachable.
- Compute next version.
- Validate `agents-md` manifest shape.
- Report missing token, remote, files, and manifest problems clearly.

Tests:

- Ready state reports current and next version.
- No published versions reports next `1.0.0`.
- Missing remote fails.
- Missing `AGENTS.md` fails.
- Invalid manifest fails with field-level message.
- JSON output includes problems array.

### Slice 9: CLI Write Client

Implement:

- Package lookup.
- Package creation.
- Draft creation and reuse handling.
- Artifact upload.
- Publish.
- Review submit.
- Approval status.
- Version lookup by package coordinates and SemVer.
- `APPROVAL_REQUIRED` publish error handling.
- Structured API error handling with request ID propagation.

Tests:

- `httptest.Server` verifies method, URL, auth header, body, and response parsing for each method.
- API error response includes code, message, status, and request ID.
- Draft lookup distinguishes reusable draft versions from immutable published versions.
- Publish approval failure returns structured `APPROVAL_REQUIRED`.

### Slice 10: Push

Implement:

- Ensure package exists or create it.
- Create or reuse selected draft version.
- Generate/update manifest allowed fields.
- Upload manifest first.
- Upload `AGENTS.md` second.
- Publish or submit for review.
- Support `--json`.

Tests:

- New package publishes `1.0.0` when no approval required.
- Existing package publishes next patch by default.
- `--minor`, `--major`, and `--version` select expected versions.
- Existing published selected version fails with suggestion.
- Existing draft selected version is reused.
- Approval required submits for review in default mode.
- Default mode does not submit for review on non-`APPROVAL_REQUIRED` publish errors.
- `--publish` fails when approval blocks publish.
- Upload order is manifest then `AGENTS.md`.
- `AGENTS.md` content is not rewritten.

### Slice 11: User-Facing Docs

Implement docs after code behavior is in place:

- New `docs-site/cli/init.md`.
- New `docs-site/cli/download.md`.
- New `docs-site/cli/clone.md`.
- New `docs-site/cli/remote.md`.
- New `docs-site/cli/push.md`.
- Remove or replace `docs-site/cli/fetch.md`.
- Update CLI sidebar from `fetch` to `download`.
- Update raw artifact docs to trailing-selector URLs.
- Update publishing docs to show ergonomic CLI first and raw API second.

## Acceptance Criteria

A consumer can retrieve `AGENTS.md` without learning Trove internals:

```bash
trove download nwks/platform/agent-defaults AGENTS.md --output AGENTS.md
```

Result:

- `AGENTS.md` is written.
- No `.trove.lock.yaml` is written.
- No `trove.yaml` is written.
- No `.trove/config.yaml` is written.

The equivalent private raw URL works with curl and a bearer token:

```bash
curl -L -H "Authorization: Bearer $TROVE_TOKEN" \
  https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md
```

A maintainer can start from a published version and prepare a new draft:

```bash
trove clone nwks/platform/agent-defaults@stable
cd agent-defaults
trove status
trove push --patch
```

A new user can initialize and publish:

```bash
trove init agents-md --remote https://trove.company.com/nwks/platform/agent-defaults --maintainer-team platform-engineering --yes
trove push
```

If publishing succeeds immediately, output resembles:

```text
Published nwks/platform/agent-defaults@1.0.0
Install with:
  trove install nwks/platform/agent-defaults@stable
```

If review is required, output resembles:

```text
Uploaded nwks/platform/agent-defaults@1.0.0
Submitted for review:
  https://trove.company.com/reviews/...
```

`trove fetch` is not supported:

```bash
trove fetch nwks/platform/agent-defaults AGENTS.md
```

Expected result:

```text
unknown subcommand: fetch
```

No user should need to know about draft APIs, artifact upload endpoints, review internals, or manual `trove.yaml` construction for the standard `AGENTS.md` case.
