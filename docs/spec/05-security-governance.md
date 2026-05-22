# Security And Governance

## Authentication

MVP supports:

- OIDC login for humans.
- Scoped opaque bearer tokens for agents, CI, and CLI use.
- Explicitly gated dev/static auth mode for local development only.
- Anonymous read only for resources with effective `public` visibility.

Production deployments must use OIDC for human login. Dev/static auth must require explicit configuration and must not be enabled by default.

OIDC integration is standards-based. Configure issuer URL, client ID, client secret, redirect URL, and scopes; do not bake provider-specific assumptions into MVP auth. Microsoft Entra, Okta, Authentik, Dex, and other OIDC providers should work through configuration.

## Authorization

RBAC roles:

| Role | Capabilities |
|---|---|
| `reader` | Read packages allowed by visibility and membership |
| `maintainer` | Create packages and draft versions in allowed namespaces |
| `reviewer` | Review and approve submitted versions |
| `namespace-admin` | Manage namespace settings and maintainers |
| `org-admin` | Manage organization settings, namespaces, and users |
| `system-admin` | Installation-wide administration |

Self-approval is disabled by default. Publishing requires one human approval in the MVP unless namespace/package policy later raises the requirement.

## Visibility Rules

- Raw URLs require auth by default.
- Effective visibility is the most restrictive value across org, namespace, package, and version.
- Public namespaces can contain public packages whose raw artifacts are anonymously readable.
- Private packages inside public namespaces remain private.
- Anonymous users can read public package metadata, manifests, archives, raw artifacts, and resolve responses.
- Anonymous users cannot read audit events, review details, drafts, non-public versions, token metadata, user/team membership, or administrative settings.
- Audit all writes and administrative reads.

Visibility order from most restrictive to least restrictive:

```text
private > internal > public
```

## API Tokens

MVP API tokens are opaque bearer tokens stored server-side as hashes.

Required token fields:

- token ID
- token hash
- display name
- actor user or service account
- scopes
- optional org, namespace, or package restriction
- created timestamp
- optional expiration timestamp
- last used timestamp
- revoked timestamp

Initial scopes:

- `package:read`
- `package:write`
- `version:publish`
- `review:write`
- `namespace:admin`
- `org:admin`

Tokens used by CI or agents should be scoped to the smallest required org, namespace, or package.

## Review Workflow

```text
draft -> submit -> automated checks -> human review -> approved -> published
```

Automated checks before approval or publishing:

- manifest schema validation
- artifact path validation
- broken internal link checks
- duplicate package detection
- required metadata checks
- markdown linting
- secret scanning
- unsafe instruction scanning
- dependency and compatibility checks

Blocking checks for MVP publishing:

- manifest schema validation
- route/package/version consistency
- artifact path validation
- configured file/package size limits
- required metadata checks
- high-confidence secret scanning
- high-risk unsafe instruction scanning

Warning-only checks for MVP publishing:

- broken external links
- markdown linting
- missing or unknown compatibility metadata

Review mutation rules:

- Submitted versions can be returned to draft for changes.
- Any artifact or manifest change after submission invalidates existing approvals and automated check results.
- Approved content cannot change before publishing without returning to draft and repeating checks/review.

Publishing performs:

- package digest computation
- per-file digest computation
- immutable content lock
- channel alias update
- audit event write
- webhook emission when configured

Publishing actor rules:

- A maintainer publishes after required approvals and blocking checks pass.
- Reviewers approve or request changes; approval does not publish automatically.
- The same actor cannot both approve and publish unless they are publishing a version approved by a different reviewer.

## Secret Scanning

Publishing must block known high-confidence secrets:

- private keys
- API tokens
- passwords
- cloud provider keys
- accidentally embedded credentials

## Unsafe Instruction Scanning

Flag or block high-risk instruction patterns:

- bypass security controls
- exfiltrate secrets
- ignore system or developer instructions
- disable tests without approval
- commit directly to protected branches
- approve own changes
- remove audit trails

## Audit Events

Audit events must include:

- actor ID
- action name
- target resource
- timestamp
- request ID
- before/after metadata when safe
- remote address or token identity

Required audited actions:

- create/update/delete org, namespace, package, version, artifact, token, team, membership, policy
- submit, approve, reject, publish, deprecate, yank
- channel alias update
- auth failures for sensitive operations
