# Manifest And Lockfile

## Package Manifest

Every package version must contain `trove.yaml`.

```yaml
apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: companyx
  namespace: platform
  name: agent-backend
  displayName: Backend Agent Defaults
  description: Default agent instructions, skills, and commands for backend services.
  labels:
    language: golang
    framework: chi
    maturity: production
  annotations:
    owner: platform-engineering
spec:
  version: 1.0.0
  license: internal
  visibility: private
  lifecycle: draft
  compatibility:
    tools:
      - name: opencode
        version: ">=0.6.0 <2.0.0"
    models:
      - family: gpt
        minContextWindow: 128000
    runtimes:
      - linux
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
    - path: skills/backend-api/SKILL.md
      type: skill
      required: false
      targetPath: .opencode/skills/backend-api/SKILL.md
  dependencies: []
  updatePolicy:
    breakingChangeRequiresManualApproval: true
  links:
    docs: https://docs.company.com/agent-backend
```

## Required Manifest Fields

- `apiVersion`
- `kind`
- `metadata.org`
- `metadata.namespace`
- `metadata.name`
- `metadata.displayName`
- `metadata.description`
- `spec.version`
- `spec.visibility`
- `spec.lifecycle`
- `spec.artifacts`

Package references in manifests must use full `org/namespace/package` form. Contextual shorthand is not supported in the MVP.

At publish time, `metadata.description` must be non-empty.

## Manifest Validation

Publishing must fail when:

- `metadata.org`, `metadata.namespace`, or `metadata.name` do not match the route/package.
- `spec.version` does not match the version being published.
- an artifact path escapes the package root.
- an artifact path is duplicated.
- a required artifact is missing.
- `metadata.description` is empty.
- an artifact type is unknown.
- compatibility metadata is malformed.
- dependency references are malformed.
- links are syntactically invalid.
- secrets or blocked unsafe instructions are detected.
- uploaded files exceed configured per-file or package size limits.

MVP dependency behavior is declare-only. The server validates dependency reference syntax and exposes dependency metadata, but does not require dependency packages to exist and does not auto-install dependencies.

## Compatibility Schema

MVP compatibility uses one generic schema shared by all agent harnesses.

Tool-specific schemas for OpenCode, Claude Code, Cursor, Copilot, and other harnesses are deferred. Tool-specific details can be carried in labels or annotations until an extension schema is designed.

```yaml
compatibility:
  tools:
    - name: opencode
      version: ">=0.6.0 <2.0.0"
  models:
    - family: gpt
      minContextWindow: 128000
  runtimes:
    - linux
    - darwin
    - kubernetes
```

Rules:

- `tools[].name` is a lowercase slug.
- `tools[].version` is a SemVer range.
- `tools[].minVersion` may be accepted as input shorthand and normalized to `version: ">=<minVersion>"`.
- `models[].family` is a lowercase family name such as `gpt`, `claude`, or `gemini`.
- `models[].minContextWindow` is optional and expressed in tokens.
- `runtimes[]` is a list of named lowercase strings such as `linux`, `darwin`, `windows`, `kubernetes`, or `docker`.
- Missing compatibility metadata evaluates to `unknown`.

## Path Rules

- Paths are relative to the package root.
- Paths use `/` separators.
- Paths must not be absolute.
- Paths must not contain `..` segments.
- Paths must not contain `@`; raw artifact URLs reserve `@` for selectors.
- Paths must not point outside the package archive.

If an artifact omits `targetPath`, the install target defaults to the artifact `path`.

## Upload Formats

MVP package uploads support:

- `.tar.gz`
- `.zip`

Uploads are unpacked into individual artifact files. The original uploaded archive bytes are not the package identity and are not used as the package digest.

## Size Limits

MVP implementations must enforce configurable upload limits.

Default limits:

- max artifact file size: 10 MiB
- max unpacked package size: 100 MiB
- max artifact count per package version: 1,000

These defaults can be raised by configuration, but Postgres-backed blob storage should keep conservative defaults until object storage is added.

## Project Lock File

Projects that install packages may pin them with `.trove.lock.yaml`.

```yaml
apiVersion: trove.io/v1
kind: TroveLock
generatedBy:
  name: trove
  version: 0.1.0
  generatedAt: "2026-05-21T00:00:00Z"
project:
  org: companyx
  name: payments-api
  repo: https://git.company.com/payments/payments-api
installs:
  - package: companyx/platform/agent-backend
    requestedSelector: latest
    version: 1.0.0
    digest: sha256:abc123
    installedAt: "2026-05-21T00:00:00Z"
    artifacts:
      - source: AGENTS.md
        target: AGENTS.md
        digest: sha256:def456
```

## Lock File Rules

- Lock files pin exact versions and digests.
- Lock files record the generating tool name, tool version, and generation timestamp.
- Lock file package references must use full `org/namespace/package` form.
- Update checks compare the pinned version and digest against server-side selectors.
- Lock files are project-owned; the registry does not directly edit downstream repositories by default.
- `trove install` must not overwrite an existing different target file unless the caller passes an explicit overwrite option.
- CLI writes canonical YAML with stable ordering when it creates or updates the lockfile.
- Comments and hand formatting in `.trove.lock.yaml` are not preserved by CLI rewrites.
