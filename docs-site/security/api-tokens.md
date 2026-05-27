# API Tokens

## Why

Agents, CI systems, and CLI tools need to interact with Trove without a browser session. API tokens provide machine access with revocation, auditing, and least-privilege scopes.

## How

### Token Fields

| Field | Type | Description |
|---|---|---|
| `id` | UUID | Token identifier |
| `tokenHash` | string | Hashed token value (never returned in API responses) |
| `displayName` | string | Human-readable label |
| `actorUserId` | UUID | The user who created this token |
| `actorServiceAccount` | string | Optional service account name |
| `scopes` | array | List of granted scopes |
| `orgId` | UUID | Optional resource metadata recorded with the token |
| `namespaceId` | UUID | Optional resource metadata recorded with the token |
| `packageId` | UUID | Optional resource metadata recorded with the token |
| `createdAt` | timestamp | When the token was created |
| `expiresAt` | timestamp | Optional: when the token expires |
| `lastUsedAt` | timestamp | When the token was last used |
| `revokedAt` | timestamp | When the token was revoked (null if active) |

### Creating a Token

Only authenticated users can create tokens:

```bash
POST /api/v1/tokens
Authorization: Bearer <session-or-token>
Content-Type: application/json

{
  "displayName": "Production CI",
  "scopes": ["package:read", "version:publish"],
  "orgId": "org-nwks-001",
  "expiresAt": "2027-01-01T00:00:00Z"
}
```

Response:

```json
{
  "id": "tok-abc123",
  "displayName": "Production CI",
  "scopes": ["package:read", "version:publish"],
  "token": "01902f1e-9c7a-7f4a-8a31-8f2b2d9c1a44",
  "createdAt": "2026-05-23T00:00:00Z"
}
```

**Important:** The `token` field is shown only once at creation. Store it securely.

### Using a Token

Include the token in the `Authorization` header:

```bash
GET /api/v1/packages/nwks/platform/agent-backend
Authorization: Bearer 01902f1e-9c7a-7f4a-8a31-8f2b2d9c1a44
```

### Default Scopes

If no scopes are specified, the token defaults to `package:read`:

```bash
POST /api/v1/tokens
{
  "displayName": "Read-only token"
}
# → scopes: ["package:read"]
```

### Revoking a Token

```bash
POST /api/v1/tokens/{tokenId}/revoke
Authorization: Bearer <session-or-token>
```

Response:

```json
{
  "status": "revoked"
}
```

Revoked tokens are immediately invalid. The `lastUsedAt` and `revokedAt` timestamps are recorded for auditing.

### Token Best Practices

| Practice | Reason |
|---|---|
| Set an expiration date | Limits exposure if a token is leaked |
| Restrict to the smallest scope | Least-privilege access reduces blast radius |
| Use separate tokens for CI and humans | Easier to revoke one without affecting the other |
| Rotate tokens periodically | Reduces the window of exposure |

### Token Lifecycle

```
Created → Active → (Last used tracking) → Revoked or Expired
```

- **Created**: Token is generated and the hash is stored
- **Active**: Token can be used for API requests
- **Last used**: Updated on each successful request
- **Revoked**: Token is permanently invalid
- **Expired**: Token is invalid after `expiresAt`

### Auditing

All token usage is audited:

- Token creation (who created it, what scopes)
- Token usage (last used timestamp)
- Token revocation (who revoked it, when)

### Example: CI Token

```bash
POST /api/v1/tokens
{
  "displayName": "agent-backend CI",
  "scopes": ["package:read", "package:write", "version:publish"],
  "expiresAt": "2026-12-31T00:00:00Z"
}
```

Grant only the scopes the automation needs. Resource ID fields are stored as token metadata in the current implementation; route authorization is enforced by scopes.

### Next Steps

- Learn about [Authentication](/security/authentication) for human login
- Understand [RBAC & Scopes](/security/rbac-scopes) for access control
