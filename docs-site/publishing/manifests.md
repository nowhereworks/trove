# Manifests

## Why

Every package needs a manifest that declares what it contains, who maintains it, what tools it works with, and how its artifacts should be installed. The manifest (`trove.yaml`) is the package's identity — it is included in the version digest and validated before publishing.

## How

### Full Example

```yaml
apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: nwks
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
  license: internal
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
  maintainers:
    - team: platform-engineering
  links:
    docs: https://docs.nwks.com/agent-backend
```

### Required Fields

| Field | Type | Description |
|---|---|---|
| `apiVersion` | string | Must be `trove.io/v1` |
| `kind` | string | Must be `AgentArtifactPackage` |
| `metadata.org` | string | Organization slug (e.g., `nwks`) |
| `metadata.namespace` | string | Namespace slug (e.g., `platform`) |
| `metadata.name` | string | Package slug (e.g., `agent-backend`) |
| `metadata.displayName` | string | Human-readable name |
| `metadata.description` | string | Non-empty description (required at publish) |
| `spec.lifecycle` | string | Current lifecycle state |
| `spec.artifacts` | array | List of artifact declarations |
| `spec.maintainers` | array | At least one maintainer (required at publish) |

### Optional Fields

| Field | Type | Description |
|---|---|---|
| `metadata.labels` | object | Key-value labels for search and filtering |
| `metadata.annotations` | object | Key-value metadata for tooling |
| `spec.license` | string | License identifier |
| `spec.compatibility` | object | Tool, model, and runtime constraints |
| `spec.dependencies` | array | Declare-only dependency references |
| `spec.links` | object | Documentation and related URLs |

### Artifact Declaration

Each artifact in `spec.artifacts`:

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `path` | string | Yes | — | Relative path within the package archive |
| `type` | string | Yes | — | One of the 10 artifact types |
| `required` | boolean | No | `false` | Whether `trove install` includes it by default |
| `targetPath` | string | No | Same as `path` | Where the file is installed on disk |

### Validation Rules

Publishing fails when:

| Rule | Error |
|---|---|
| `metadata.org`, `namespace`, or `name` don't match the route | `INVALID_MANIFEST` |
| Artifact path escapes package root or contains `..` | `INVALID_MANIFEST` |
| Duplicate artifact paths | `INVALID_MANIFEST` |
| Required artifact file missing | `INVALID_MANIFEST` |
| `metadata.description` is empty | `INVALID_MANIFEST` |
| No maintainer declared | `INVALID_MANIFEST` |
| Unknown artifact type | `INVALID_MANIFEST` |
| Malformed compatibility metadata | `INVALID_MANIFEST` |
| Malformed dependency references | `INVALID_MANIFEST` |
| Invalid link URLs | `INVALID_MANIFEST` |
| Secrets or blocked unsafe instructions detected | `SECRET_DETECTED` / `UNSAFE_INSTRUCTION` |
| File exceeds size limits | `ARTIFACT_TOO_LARGE` |

### Path Rules

- Paths are relative to the package root
- Use `/` separators
- Must not be absolute
- Must not contain `..` segments
- Must not point outside the package archive

### Next Steps

- See the full [Upload & Publish Flow](/publishing/upload-publish-flow)
- Learn about [Security Scanning](/publishing/security-scanning) that validates manifests
