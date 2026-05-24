# AGENTS.md CLI Publishing Plan

## Goal

Make publishing repo-root `AGENTS.md` files to Trove feel natural, ergonomic, and low-bureaucracy for developers, platform teams, and citizen developers.

The CLI should hide Trove's internal draft, manifest, upload, review, and publish workflow behind a simple Git-like flow:

```bash
trove init agents-md
trove remote add origin https://trove.company.com/nwks/platform/agent-defaults
trove status
trove push
```

Also support the convenience alias:

```bash
trove init --agents-md
```

Do not use `trove agent init`; that name conflicts with future subagent or `agents/` artifact workflows.

## Command Contract

This plan is the `AGENTS.md` specialization of the Trove-wide rules in [`12-command-semantics.md`](12-command-semantics.md). Keep package references, selector defaults, raw URL behavior, install semantics, and worktree semantics consistent with that document.

## Retrieval Contract

A published `AGENTS.md` can be retrieved as a raw artifact, downloaded as one file, installed into a consuming repository, or cloned into an editable package worktree.

### Raw URL

The raw URL form is:

```text
/raw/{org}/{namespace}/{package}/AGENTS.md[@selector]
```

Examples:

```text
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@v1.0.3
```

If the selector is omitted, the server resolves `stable`. Alias or omitted-selector URLs redirect to exact immutable URLs:

```text
GET /raw/nwks/platform/agent-defaults/AGENTS.md
302 /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3

GET /raw/nwks/platform/agent-defaults/AGENTS.md@1.0.3
200 AGENTS.md bytes
```

Private and internal packages require a bearer token:

```bash
curl -L -H "Authorization: Bearer $TROVE_TOKEN" \
  https://trove.company.com/raw/nwks/platform/agent-defaults/AGENTS.md@stable
```

### Download For Immediate Use

Use `trove download` when the caller wants the `AGENTS.md` file but does not want Trove project metadata.

```bash
trove download nwks/platform/agent-defaults AGENTS.md --output AGENTS.md
trove download nwks/platform/agent-defaults@v1.0.3 AGENTS.md --output AGENTS.md
```

Behavior:

- Omitted selector resolves to `stable`.
- Only `AGENTS.md` is written.
- No `.trove.lock.yaml`, `trove.yaml`, or `.trove/config.yaml` is written.
- Existing files are not overwritten unless overwrite is explicit.

### Install Into A Consuming Repo

Use `trove install` when a downstream project adopts the package and wants update tracking.

```bash
trove install nwks/platform/agent-defaults --target .
```

Behavior:

- Resolve the package to an exact published version.
- Read the manifest.
- Install required artifacts, including `AGENTS.md`.
- Write `.trove.lock.yaml` with exact version and digest pins.
- Do not write publishing metadata.

### Clone For Maintainer Edits

Use `trove clone` when a maintainer wants to edit and later publish a new version.

```bash
trove clone nwks/platform/agent-defaults@stable
cd agent-defaults
trove status
trove push --patch
```

Behavior:

- Resolve the selected published version.
- Download `trove.yaml` and all manifest-listed artifacts needed for the editable package checkout.
- Write `.trove/config.yaml` with the remote and default publishing settings.
- Prepare future `trove status`, `trove pull`, and `trove push` commands.
- Treat edits as changes for a new draft version; never mutate the cloned published version.

`trove pull` refreshes the editable checkout from the configured remote and must not overwrite local changes silently.

### `trove init agents-md`

Creates or adopts a repo-root `AGENTS.md` and sets up Trove publishing metadata.

Behavior:

- If `AGENTS.md` exists, reuse it.
- If `AGENTS.md` is missing, create a starter file.
- If `trove.yaml` is missing, create it.
- If `.trove/config.yaml` is missing, create it.
- If run again, be idempotent and non-destructive.
- If `--remote` is provided, configure the remote immediately.

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
| `--maintainer-team <team>` | Set the manifest maintainer team |
| `--force` | Overwrite generated metadata only; never overwrite `AGENTS.md` |

### `trove remote add origin <remote>`

Stores where this local `AGENTS.md` publishes.

Accepted remote forms:

```bash
https://trove.company.com/nwks/platform/agent-defaults
nwks/platform/agent-defaults
```

If only a package ref is provided, resolve the server from:

1. `TROVE_SERVER_URL`
2. Existing `.trove/config.yaml`
3. Otherwise return a clear error asking for a full URL

Also add:

```bash
trove remote list
trove remote remove origin
```

### `trove status`

Shows what will happen before push:

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

Status should report missing token, remote, `AGENTS.md`, or manifest problems clearly.

### `trove push`

Runs the full publishing workflow.

Default behavior:

- Resolve the configured remote package.
- Create the package if it does not exist.
- Pick the next patch version automatically.
- Generate or update `trove.yaml` for that version.
- Create a draft version.
- Upload `trove.yaml` first.
- Upload `AGENTS.md`.
- Try to publish.
- If approval is required, submit for review and print the review URL instead of failing.

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

## Local Files

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

### `trove.yaml`

This remains the actual package manifest uploaded to Trove.

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

`trove push` may update generated fields in `trove.yaml`, especially:

- `metadata.org`
- `metadata.namespace`
- `metadata.name`
- `spec.version`
- `spec.channel`
- `spec.visibility`
- `spec.lifecycle`

It must not rewrite `AGENTS.md`.

## Version Selection

Default publishing version selection follows [`12-command-semantics.md`](12-command-semantics.md). For the standard `AGENTS.md` case, versioning should stay invisible unless the user needs control.

Rules:

- No published versions: `1.0.0`
- Existing latest `1.0.0`: next default is `1.0.1`
- `--minor`: `1.1.0`
- `--major`: `2.0.0`
- `--version`: exact user-provided version
- If the selected version already exists as draft, reuse and update it.
- If the selected version already exists as published, fail and suggest the next patch version.

## API Work Needed

Add write-side CLI client methods:

- `GetPackage`
- `CreatePackage`
- `CreateDraftVersion`
- `UploadArtifact`
- `PublishVersion`
- `SubmitReview`
- `GetApprovalStatus`

Initially continue using `TROVE_SERVER_URL` and `TROVE_TOKEN`. A polished `trove login` command can come later.

## Backend Fix Needed

Before `trove push` can reliably support review-gated publishing, fix review route handling.

Current issue:

- Review handlers pass the `{version}` route param, such as `1.0.0`, into review service methods that expect a package version UUID.
- Publish approval checks also call approval status with the SemVer string.
- Approval-gated publishing is therefore likely broken for real PostgreSQL-backed usage.

Implementation fix:

- Add a query to resolve package version ID by `org`, `namespace`, `package`, and `version`.
- In review submit, status, and publish handlers, resolve the SemVer route to the package version UUID before calling the review service.
- Add tests for submit, approve, approval status, and publish with approval required.
- Add scope enforcement for review routes to match docs: `review:write`.

## Implementation Slices

1. Add trailing-selector raw URL support.
2. Add `trove download` for single-file retrieval.
3. Add `trove clone` and `trove pull` for editable worktrees.
4. Add local project config support.
5. Add CLI write client methods.
6. Add `trove init agents-md`.
7. Add `trove remote`.
8. Add `trove status`.
9. Add `trove push`.
10. Fix review backend route-to-version-ID handling.
11. Update user-facing docs.

### Slice 1: Raw URL Retrieval

Implement:

- `/raw/{org}/{namespace}/{package}/AGENTS.md[@selector]`.
- Omitted selector resolves `stable`.
- Alias and omitted-selector URLs redirect to exact immutable URLs.
- Exact raw URLs return `AGENTS.md` bytes with digest `ETag`.
- Private and internal reads require bearer auth.

### Slice 2: Download

Implement:

- `trove download <package-ref> AGENTS.md`.
- `--output <file>`.
- Omitted package selector resolves `stable`.
- Existing files are not overwritten unless overwrite is explicit.
- No lockfile or publishing metadata writes.
- `--json` metadata output when not streaming bytes.

### Slice 3: Clone And Pull

Implement:

- `trove clone <package-ref> [dir]`.
- `trove pull` from a cloned worktree.
- Download `trove.yaml`, `AGENTS.md`, and `.trove/config.yaml`.
- Detect local changes before overwriting files.
- Prepare the checkout for `trove status` and `trove push`.

### Slice 4: Local Project Config

Implement:

- Read and write `.trove/config.yaml`.
- Remote URL parser.
- Package ref parser without selector.
- Tests for parsing and round-trip config.

### Slice 5: CLI Write Client

Implement:

- Package, draft, upload, publish, and review API calls.
- Structured API error handling.
- Tests with `httptest.Server`.

### Slice 6: Init

Implement:

- Create or adopt `AGENTS.md`.
- Create or update `trove.yaml`.
- Create `.trove/config.yaml`.
- Add CLI routing and tests.

### Slice 7: Remote

Implement:

- `trove remote add`
- `trove remote list`
- `trove remote remove`
- Full URL and short package ref support.
- No network call required for `remote add`.

### Slice 8: Status

Implement:

- Load local state.
- Query remote package if reachable.
- Compute next version.
- Report missing token, remote, files, and manifest problems clearly.

### Slice 9: Push

Implement:

- Ensure package exists or create it.
- Create or reuse draft.
- Upload manifest first.
- Upload `AGENTS.md`.
- Publish or submit for review.
- Support `--json`.

### Slice 10: Review Backend

Implement:

- Route-to-package-version-ID lookup.
- Approval-gated publish tests.
- Scope enforcement for review routes.

### Slice 11: Docs

Implement:

- New `docs-site/cli/init.md`.
- New `docs-site/cli/download.md`.
- New `docs-site/cli/clone.md`.
- New `docs-site/cli/remote.md`.
- New `docs-site/cli/push.md`.
- Update CLI sidebar.
- Update publishing docs to show ergonomic CLI first and raw API second.

## Acceptance Criteria

A consumer can retrieve `AGENTS.md` without learning Trove internals:

```bash
trove download nwks/platform/agent-defaults AGENTS.md --output AGENTS.md
```

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

A new user can run:

```bash
trove init agents-md --remote https://trove.company.com/nwks/platform/agent-defaults
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

No user should need to know about draft APIs, artifact upload endpoints, or manual `trove.yaml` construction for the standard `AGENTS.md` case.
