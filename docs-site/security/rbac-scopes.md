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
| `namespace:write` | Manage namespace settings |
| `org:write` | Manage organization settings |

### Scope-Protected Routes

Write API routes require specific scopes:

| Route | Required Scope |
|---|---|
| `POST /api/v1/packages/{org}/{namespace}/{package}/versions` | `package:write` |
| `PUT /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/artifacts/*` | `package:write` |
| `POST /api/v1/packages/{org}/{namespace}/{package}/versions/{version}/publish` | `version:publish` |
| `POST /api/v1/reviews/{org}/{namespace}/{package}/versions/{version}/submit` | `review:write` |
| `POST /api/v1/reviews/{reviewId}/approve` | `review:write` |
| `POST /api/v1/orgs` | `org:write` |
| `POST /api/v1/orgs/{org}/namespaces` | `namespace:write` |
| `POST /api/v1/packages` | `package:write` |

### Token Resource Metadata

Token creation accepts `orgId`, `namespaceId`, and `packageId` fields and stores them with the token for auditing and future policy work. The current authorization path enforces scopes on protected routes; it does not enforce resource ID matching.

### Effective Permissions

A user's effective permissions are determined by:

1. Their role (reader, maintainer, reviewer, etc.)
2. The token's scopes (if using a token)
3. The resource's visibility (private, internal, public)

### Example: Creating a Scoped Token

```bash
# Create a token that can read and write packages
POST /api/v1/tokens
Authorization: Bearer <session>

{
  "displayName": "CI token",
  "scopes": ["package:read", "package:write"],
  "expiresAt": "2027-06-01T00:00:00Z"
}
```

### Next Steps

- Learn about [Authentication](/security/authentication) modes
- See how to manage [API Tokens](/security/api-tokens)
