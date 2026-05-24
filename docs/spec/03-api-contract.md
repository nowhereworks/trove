# API Contract

## API Principles

- Agent-facing APIs are first-class and must not depend on the web UI.
- Selectors resolve to exact immutable versions before artifacts are served.
- Private and internal resources require authentication.
- Public resources are anonymously readable only when explicitly marked public.
- JSON fields use `camelCase`.
- List endpoints use cursor pagination with `limit` and `cursor` query parameters.
- Every response includes `X-Request-Id`; error bodies include the same request ID.
- Errors use one JSON shape across APIs.

## Routing

Use `chi` for HTTP routing and middleware in the MVP.

Registry-style package references stay in one route segment where applicable:

```text
/api/v1/resolve/{org}/{namespace}/{package}@{selector}
```

Handlers parse the final path segment into `{package}` and `{selector}`. The selector separator is the last `@` in that segment. Package slugs cannot contain `@`.

Raw artifact URLs use the trailing selector syntax defined in [`12-command-semantics.md`](12-command-semantics.md):

```text
/raw/{org}/{namespace}/{package}/{artifactPath...}[@selector]
```

If the selector is omitted, raw artifact URLs resolve `stable`.

## Authentication

Supported MVP authentication modes:

- Browser session from OIDC login.
- API token for CLI, CI, and agents.
- Dev/static auth mode only when explicitly enabled for local development.
- Anonymous read for resources with effective `public` visibility.

Raw URLs require authentication by default. Public namespaces/packages can opt into anonymous reads.

API tokens are opaque scoped bearer tokens. Tokens must be looked up server-side so they can be revoked and audited.

Anonymous users may read public package metadata, manifests, archives, raw artifacts, and resolve responses. Anonymous users may not read audit events, review details, drafts, non-public versions, or administrative metadata.

## Public And Agent APIs

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/resolve/{org}/{namespace}/{package}@{selector}` | visibility-dependent | Resolve a selector to an exact version |
| `GET` | `/api/v1/search/packages` | visibility-dependent | Search discoverable packages |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}` | visibility-dependent | Get package metadata |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/manifest` | visibility-dependent | Get the package manifest |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.tar.gz` | visibility-dependent | Download package archive |
| `GET` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/archive.zip` | visibility-dependent | Download package archive |
| `GET` | `/raw/{org}/{namespace}/{package}/{path...}@{version}` | visibility-dependent | Fetch one immutable artifact file |
| `GET` | `/raw/{org}/{namespace}/{package}/{path...}@{selector}` | visibility-dependent | Resolve selector, then redirect to exact file |
| `GET` | `/raw/{org}/{namespace}/{package}/{path...}` | visibility-dependent | Resolve omitted selector as `stable`, then redirect to exact file |
| `POST` | `/api/v1/updates/check` | token/session | Check for newer compatible releases |
| `POST` | `/api/v1/projects/report` | token/session | Report project lockfile/adoption summary |
| `POST` | `/api/v1/compatibility/check` | token/session | Check if a package is compatible with a target |

List endpoints return cursor-paginated envelopes:

```json
{
  "items": [],
  "nextCursor": "opaque-cursor-or-null"
}
```

Clients may pass `limit` and `cursor`. Servers may cap `limit`.

Package search defaults to published active versions only. Authenticated callers may use explicit filters to include deprecated content when authorized. Draft, review, yanked, and archived versions are excluded from normal search results.

MVP package search filters:

- `org`
- `namespace`
- `artifactType`
- `language`
- `framework`
- `tool`
- `visibility`
- `lifecycle`

## Management APIs

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `POST` | `/api/v1/orgs` | admin | Create organization |
| `POST` | `/api/v1/orgs/{org}/namespaces` | org admin | Create namespace |
| `POST` | `/api/v1/packages` | maintainer | Create package |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions` | maintainer | Create draft version |
| `PUT` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/{path...}` | maintainer | Upload draft artifact |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/submit` | maintainer | Submit for review |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/approve` | reviewer | Approve version |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/request-changes` | reviewer | Request changes with a simple comment |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish` | maintainer | Publish immutable version |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/deprecate` | maintainer | Deprecate version |
| `POST` | `/api/v1/packages/{org}/{namespace}/{package}/versions/{version}/yank` | admin/maintainer | Yank version |

Create endpoints return `201 Created` with the created resource JSON and `Location` when a canonical URL exists.

Synchronous action endpoints such as `submit`, `approve`, `publish`, `deprecate`, and `yank` return `200 OK` with the updated resource JSON.

Review action request bodies may include a simple `comment` string. MVP review comments are version-level only; threaded and line-level comments are deferred.

## Resolve Response

```json
{
  "org": "companyx",
  "namespace": "platform",
  "package": "agent-backend",
  "selector": "stable",
  "resolvedVersion": "1.0.0",
  "digest": "sha256:abc123",
  "manifestUrl": "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/manifest",
  "archiveUrl": "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/archive.tar.gz"
}
```

## Raw File Behavior

For exact versions:

- return `200 OK` with artifact bytes when authorized
- include `ETag` with the artifact digest
- include `Cache-Control: public, max-age=31536000, immutable` for public exact versions
- include `Cache-Control: private, max-age=31536000, immutable` for private exact versions

For aliases:

- resolve selector to exact version
- return `302 Found` to the exact raw URL
- include `Cache-Control: no-cache` for alias responses

Alias raw URLs must not serve artifact bytes directly in the MVP. Redirecting keeps caching and debugging behavior explicit.

Artifact paths must not contain `@` because `@` is reserved for raw URL selectors.

## Archive Behavior

MVP archive downloads are generated on demand from stored artifact files and blobs.

Rules:

- Archive entries are sorted by artifact path for deterministic output.
- Archive entry paths match manifest artifact paths.
- Archive generation must not mutate package version state.
- Exact version archive responses include `ETag` based on the package digest.
- Alias archive URLs must redirect to exact version archive URLs.
- Supported archive formats are `.tar.gz` and `.zip`.

## Update Check Request

```json
{
  "package": "companyx/platform/agent-backend",
  "currentVersion": "1.0.0",
  "currentDigest": "sha256:abc123",
  "channel": "stable",
  "strictCompatibility": false,
  "target": {
    "tool": "opencode",
    "toolVersion": "0.6.0",
    "runtime": "linux",
    "modelFamily": "gpt",
    "contextWindow": 128000
  }
}
```

## Update Check Response

```json
{
  "updateAvailable": true,
  "latestVersion": "1.1.0",
  "latestDigest": "sha256:def456",
  "compatibility": "compatible",
  "requiresManualApproval": false,
  "changelogUrl": "/api/v1/packages/companyx/platform/agent-backend/compare/1.0.0...1.1.0"
}
```

`compatibility` values are `compatible`, `incompatible`, and `unknown`. Missing or unsupported compatibility metadata returns `unknown`; it is advisory by default unless policy or client flags require strict compatibility.

When `strictCompatibility` is true, `unknown` compatibility is treated as a failure by clients and policy checks. The API still returns `compatibility: "unknown"`; callers decide whether to block based on strict mode and policy.

Tool compatibility uses SemVer range matching. Runtime compatibility uses exact named-string matching. Model compatibility compares model family and optional minimum context window.

## Error Shape

Every response includes an `X-Request-Id` header.

For errors, the JSON body includes the same request ID:

```json
{
  "error": {
    "code": "PACKAGE_NOT_FOUND",
    "message": "Package companyx/platform/agent-backend was not found.",
    "requestId": "req_123"
  }
}
```

Common codes:

- `UNAUTHORIZED`
- `FORBIDDEN`
- `PACKAGE_NOT_FOUND`
- `VERSION_NOT_FOUND`
- `ARTIFACT_NOT_FOUND`
- `INVALID_MANIFEST`
- `VERSION_ALREADY_EXISTS`
- `VERSION_IMMUTABLE`
- `REVIEW_REQUIRED`
- `APPROVAL_REQUIRED`
- `POLICY_VIOLATION`
- `YANKED_VERSION`
