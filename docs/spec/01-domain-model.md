# Domain Model

## Canonical Hierarchy

The MVP hierarchy is fixed:

```text
Organization
  Namespace
    Package
      Version
        Artifact Files
        Manifest
        Metadata
        Changelog
        Compatibility Rules
        Checksums
```

MVP package references use exactly one namespace level:

```text
{org}/{namespace}/{package}@{selector}
```

Nested namespaces are deferred.

Package names are unique within a namespace. The canonical package identity is always the full `org/namespace/package` tuple.

## Slugs

Organization, namespace, and package slugs use the same lowercase DNS-like format:

```text
^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$
```

Rules:

- Slugs are compared case-sensitively after requiring lowercase input.
- Display names can contain spaces, punctuation, and uppercase letters.
- Slug renames are not supported in the MVP.
- Display names and descriptions can change after publish; canonical slugs cannot.

## Example References

```text
companyx/platform/agent-backend@1.0.0
companyx/platform/agent-backend@latest
companyx/platform/agent-backend@v1
```

Manifests, lock files, CLI commands, and APIs must use full package references. Shorthand references like `platform/agent-backend` or `agent-backend` are not supported in the MVP.

## Artifact Types

First-class artifact types:

| Type | Example | Purpose |
|---|---|---|
| `agent-instructions` | `AGENTS.md` | Repo-level agent behavior and engineering rules |
| `skill` | `skills/api-review/SKILL.md` | Reusable focused capability for agents |
| `command` | `commands/review.md` | Slash-command-style workflow |
| `subagent` | `agents/security-reviewer.md` | Specialized agent role/persona |
| `context-pack` | `context/kubernetes.md` | Domain knowledge bundle |
| `template` | `templates/backend-api/**` | Project or artifact scaffolding |
| `rubric` | `rubrics/pr-review.md` | Review criteria |
| `policy` | `policies/model-routing.yaml` | Tool/model/runtime rules |
| `prompt` | `prompts/incident-analysis.md` | Reusable prompt pattern |
| `adapter` | `adapters/opencode.md` | Mapping for a specific agent harness |

## Version Selectors

Published MVP versions must use strict Semantic Versioning:

```text
MAJOR.MINOR.PATCH
```

Examples:

```text
1.0.0
2.3.4
```

Prerelease versions and arbitrary version strings are deferred. CLI and API inputs may include a leading `v` for selectors, but the stored version value omits `v`.

| Selector | Resolution |
|---|---|---|
| `@latest` | Newest published version |
| `@v3` | Newest version with major version `3` |
| `@v3.2` | Newest patch version in minor series `3.2` |
| `@v3.2.1` | Exact immutable version `3.2.1` |
| `@sha256:<digest>` | Exact package-version digest |

Selectors resolve server-side to exact immutable versions before content is served.

`@latest` is the only named alias. Major and minor selectors such as `v3` and `v3.2` are derived from SemVer.

`@sha256:<digest>` identifies a whole package version. Individual artifact blob digests are exposed in metadata and lock files, but are not valid package selectors.

## Lifecycle States

Package versions move through this lifecycle:

```text
draft -> review -> approved -> published -> deprecated | yanked | archived
```

Rules:

- Draft versions are mutable.
- Published versions are immutable.
- Fixes require a new version.
- Deprecated versions remain fetchable but are discouraged.
- Yanked versions are hidden from default selector resolution but remain fetchable by exact version when permitted.
- Yanked versions remain fetchable by exact version or package digest for authorized clients so locked projects stay reproducible.
- Archived packages are read-only and excluded from normal discovery.

## Visibility

Visibility can be set at organization, namespace, package, and version levels.

MVP values:

| Visibility | Meaning |
|---|---|
| `private` | Requires authenticated user or token with access |
| `internal` | Requires authentication within the owning installation |
| `public` | Raw files and public metadata are anonymously readable |

Default visibility is `private` unless an organization or namespace policy says otherwise.

Public namespaces are supported so teams can publish GitHub-like public package collections.

Effective visibility is the most restrictive value across organization, namespace, package, and version:

```text
private > internal > public
```

A package is anonymously readable only when every layer in the chain allows `public`. A private package inside a public namespace remains private.

Anonymous public reads include package metadata, manifests, archives, raw artifacts, and resolve responses. Audit events, review details, administrative metadata, drafts, and non-public versions are never anonymous in the MVP.

## Immutability

When a version is published, the system must:

- compute a package digest
- compute per-file digests
- lock artifact blobs
- prevent updates to file content, manifest content, version, and digest fields
- write an audit event
- update the `@latest` alias pointer

The `@latest` alias is mutable; exact version content is not.
