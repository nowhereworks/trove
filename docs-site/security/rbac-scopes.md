# RBAC & Scopes

## Why

Not every user should be able to publish packages, approve reviews, or manage organizations. Role-based access control (RBAC) ensures that users and tokens can only perform the actions they're authorized for. Scopes provide fine-grained control for API tokens.

## How

### RBAC Roles

| Role | Capabilities |
|---|---|
| `reader` | Read packages allowed by visibility and membership |
| `maintainer` | Create packages and draft versions in allowed namespaces |
| `reviewer` | Review and approve submitted versions |
| `namespace-admin` | Manage namespace settings and maintainers |
| `org-admin` | Manage organization settings, namespaces, and users |
| `system-admin` | Installation-wide administration |

### Token Scopes

API tokens use scopes rather than roles. Each scope grants a specific capability:

| Scope | Description |
|---|---|
| `package:read` | Read package metadata, manifests, and artifacts |
| `package:write` | Create packages, draft versions, and upload artifacts |
| `version:publish` | Publish immutable versions |
| `review:write` | Submit for review, approve, request changes |
| `namespace:admin` | Manage namespace settings |
| `org:admin` | Manage organization settings |

### Scope-Protected Routes

Write API routes require specific scopes:

| Route | Required Scope |
|---|---|
| `POST /api/v1/packages/{org}/{namespace}/{package}/versions` | `package:write` |
| `PUT /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/*` | `package:write` |
| `POST /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish` | `version:publish` |
| `POST /api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/submit` | `review:write` |
| `POST /api/v1/reviews/{reviewId}/approve` | `review:write` |
| `POST /api/v1/orgs` | `org:admin` |
| `POST /api/v1/orgs/{org}/namespaces` | `namespace:admin` |
| `POST /api/v1/packages` | `package:write` |

### Token Restrictions

Tokens can be restricted to specific resources:

| Restriction | Effect |
|---|---|
| `orgId` | Token only works for one organization |
| `namespaceId` | Token only works for one namespace |
| `packageId` | Token only works for one package |

Example: a CI token restricted to one package:

```bash
POST /api/v1/tokens
{
  "displayName": "CI for agent-backend",
  "scopes": ["package:read"],
  "packageId": "pkg-abc123"
}
```

This token can only read `nwks/platform/agent-backend`. It cannot access other packages, even in the same namespace.

### Effective Permissions

A user's effective permissions are the intersection of:

1. Their role (reader, maintainer, reviewer, etc.)
2. The token's scopes (if using a token)
3. The token's resource restrictions (if any)
4. The resource's visibility (private, internal, public)

### Example: Creating a Scoped Token

```bash
# Create a token that can only read and write in one namespace
POST /api/v1/tokens
Authorization: Bearer <session>

{
  "displayName": "Namespace CI token",
  "scopes": ["package:read", "package:write"],
  "namespaceId": "ns-def456",
  "expiresAt": "2027-06-01T00:00:00Z"
}
```

### Example: Using a Restricted Token

```bash
# This works — reading the allowed namespace
GET /api/v1/packages/nwks/platform/agent-backend
Authorization: Bearer trove_tk_restricted...

# This fails — different namespace
GET /api/v1/packages/nwks/frontend/react-defaults
Authorization: Bearer trove_tk_restricted...
# → 403 FORBIDDEN
```

### Next Steps

- Learn about [Authentication](/security/authentication) modes
- See how to manage [API Tokens](/security/api-tokens)
