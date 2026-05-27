# Authentication

## Why

Trove serves both human users and automated agents. Humans need a familiar login flow, while agents and CI systems need machine-readable credentials. Trove supports both through OIDC for humans and scoped API tokens for machines, with a gated dev mode for local development.

## How

### Authentication Modes

| Mode | Use Case | Enabled By |
|---|---|---|---|
| **OIDC** | Production human login | Configured issuer URL, client ID, and client secret |
| **API tokens** | Agents, CI, CLI | Created by authenticated users |
| **Dev mode** | Local development and tests | Explicit `auth.mode: dev` configuration |

### OIDC Login (Humans)

OIDC is standards-based and provider-neutral. Microsoft Entra, Okta, Authentik, Dex, and other OIDC providers work through configuration. Trove reads the provider's `.well-known/openid-configuration` document at startup and uses the discovered authorization, token, and UserInfo endpoints.

```yaml
auth:
  mode: oidc
  oidc:
    issuerUrl: "https://login.microsoftonline.com/tenant-id/v2.0"
    clientId: "trove"
    clientSecretRef: "TROVE_OIDC_CLIENT_SECRET"
    redirectUrl: "https://trove.nwks.com/auth/oidc/callback"
    scopes:
      - openid
      - profile
      - email
```

Login flow:

1. User visits `/auth/oidc/login`
2. Trove redirects to the OIDC provider's authorization endpoint
3. User authenticates with the provider
4. Provider redirects back to `/auth/oidc/callback` with an authorization code
5. Trove exchanges the code for tokens, creates or updates the user record
6. User receives a browser session cookie

For Microsoft Entra ID, follow the [Azure Entra ID setup guide](/security/azure-entra-id).

### API Tokens (Agents and CI)

API tokens are opaque bearer tokens stored server-side as hashes. They are created by authenticated users:

```bash
POST /api/v1/tokens
Authorization: Bearer <session-or-token>

{
  "displayName": "CI deploy token",
  "scopes": ["package:read", "package:write"],
  "expiresAt": "2027-01-01T00:00:00Z"
}
```

Response:

```json
{
  "id": "tok-abc123",
  "displayName": "CI deploy token",
  "scopes": ["package:read", "package:write"],
  "token": "trove_tk_abc123def456...",
  "createdAt": "2026-05-23T00:00:00Z"
}
```

The raw token value is shown only once at creation time. Use it in API requests:

```bash
GET /api/v1/packages/nwks/platform/agent-backend
Authorization: Bearer trove_tk_abc123def456...
```

### Dev Mode

For local development only:

```yaml
auth:
  mode: dev
  devModeEnabled: true
```

Dev mode bypasses OIDC and accepts a static token. It must not be enabled in production.

### Token Hashing

API tokens are stored as hashes, not plaintext. The hash secret is configured separately:

```yaml
auth:
  tokens:
    hashSecretRef: "TROVE_TOKEN_HASH_SECRET"
```

This means:

- Tokens cannot be recovered if the database is compromised
- Token verification happens server-side by hashing the presented token
- Tokens can be revoked at any time

### Configuration Reference

| Config Key | Type | Default | Description |
|---|---|---|---|
| `auth.mode` | string | `oidc` | `oidc` or `dev` |
| `auth.devModeEnabled` | boolean | `false` | Enable dev mode (local only) |
| `auth.oidc.issuerUrl` | string | — | OIDC provider issuer URL |
| `auth.oidc.clientId` | string | — | OIDC client ID |
| `auth.oidc.clientSecretRef` | string | — | Environment variable or secret ref |
| `auth.oidc.redirectUrl` | string | — | OIDC callback URL |
| `auth.oidc.scopes` | array | `[openid, profile, email]` | Requested OIDC scopes |
| `auth.tokens.hashSecretRef` | string | — | Secret for token hashing |

### Next Steps

- Learn about [RBAC & Scopes](/security/rbac-scopes) for access control
- See how to manage [API Tokens](/security/api-tokens)
- Configure [Azure Entra ID](/security/azure-entra-id) as an OIDC provider
