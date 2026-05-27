# Azure Entra ID Authentication

## Why

Azure Entra ID can act as Trove's OpenID Connect provider for human browser login. Trove uses the authorization code flow, discovers provider endpoints from Entra's OIDC metadata document, creates or updates a local Trove user, and then issues a Trove browser session.

Use this setup when your Trove server is deployed behind HTTPS and users should sign in with work or school accounts from your Microsoft tenant.

## How

### Prerequisites

You need:

- A deployed Trove server with a stable HTTPS URL, such as `https://trove.nwks.com`
- Access to the Microsoft Entra admin center for your tenant
- A PostgreSQL-backed Trove deployment
- A place to store the Entra client secret, such as a Kubernetes Secret or process environment variable

The examples below use `https://trove.nwks.com`. Replace it with your Trove server URL.

### 1. Create The Entra App Registration

In the Microsoft Entra admin center:

1. Open **Microsoft Entra ID**.
2. Open **App registrations**.
3. Select **New registration**.
4. Set **Name** to `Trove`.
5. Set **Supported account types** to the tenant access model you want.
6. Under **Redirect URI**, choose **Web**.
7. Enter `https://trove.nwks.com/auth/oidc/callback`.
8. Select **Register**.

Use **Single tenant** unless this Trove instance should accept users from multiple Microsoft tenants.

### 2. Copy Tenant And Client IDs

From the app registration **Overview** page, copy:

| Entra Value | Trove Use |
|---|---|
| **Directory (tenant) ID** | Used in `TROVE_OIDC_ISSUER_URL` |
| **Application (client) ID** | Used in `TROVE_OIDC_CLIENT_ID` |

The issuer URL is:

```text
https://login.microsoftonline.com/<tenant-id>/v2.0
```

For example:

```text
https://login.microsoftonline.com/00000000-1111-2222-3333-444444444444/v2.0
```

Trove reads Entra's OIDC discovery document at:

```text
https://login.microsoftonline.com/<tenant-id>/v2.0/.well-known/openid-configuration
```

Discovery is important because Entra's authorization and token endpoints are under `/oauth2/v2.0`, while its UserInfo endpoint is hosted by Microsoft Graph.

### 3. Create A Client Secret

In the app registration:

1. Open **Certificates & secrets**.
2. Select **Client secrets**.
3. Select **New client secret**.
4. Enter a description such as `trove-production`.
5. Choose an expiration period that matches your rotation policy.
6. Select **Add**.
7. Copy the secret **Value** immediately.

Store the copied secret value securely. Entra only shows it once.

### 4. Confirm Redirect URI

In **Authentication** for the app registration, confirm the **Web** platform includes exactly:

```text
https://trove.nwks.com/auth/oidc/callback
```

The redirect URI in Entra must exactly match `TROVE_OIDC_REDIRECT_URL`, including scheme, host, path, and trailing slash behavior.

### 5. Configure Trove

Set these environment variables on the Trove server:

```bash
export TROVE_AUTH_MODE=oidc
export TROVE_PUBLIC_URL=https://trove.nwks.com
export TROVE_OIDC_ISSUER_URL=https://login.microsoftonline.com/00000000-1111-2222-3333-444444444444/v2.0
export TROVE_OIDC_CLIENT_ID=11111111-2222-3333-4444-555555555555
export TROVE_OIDC_CLIENT_SECRET='<entra-client-secret-value>'
export TROVE_OIDC_REDIRECT_URL=https://trove.nwks.com/auth/oidc/callback
export TROVE_COOKIE_SECURE=true
```

Trove requests these OIDC scopes by default:

| Scope | Purpose |
|---|---|
| `openid` | Required OIDC sign-in scope |
| `profile` | Allows Entra to return the user's display name |
| `email` | Allows Entra to return the user's email address when available |

If Trove is running in Kubernetes, store the client secret in a Secret and expose it as `TROVE_OIDC_CLIENT_SECRET` to the server container.

### 6. Start Trove

Start the server after the OIDC variables are available:

```bash
./trove serve
```

At startup, Trove fetches Entra's discovery document. If discovery fails, the server reports an OIDC configuration error instead of starting with broken login URLs.

### 7. Test Browser Login

Open:

```text
https://trove.nwks.com/auth/oidc/login
```

Expected flow:

1. Trove redirects to Microsoft login.
2. The user signs in and grants requested OIDC permissions if prompted.
3. Entra redirects back to `https://trove.nwks.com/auth/oidc/callback`.
4. Trove creates or updates the local user record.
5. Trove sets a secure `trove_session` cookie and redirects to `/`.

After login, check the current session:

```bash
curl -i https://trove.nwks.com/api/v1/auth/me
```

Browser requests with the session cookie should return an authenticated user.

## Reference

### Entra Values

| Value | Where To Find It | Example |
|---|---|---|
| Tenant ID | App registration Overview, **Directory (tenant) ID** | `00000000-1111-2222-3333-444444444444` |
| Client ID | App registration Overview, **Application (client) ID** | `11111111-2222-3333-4444-555555555555` |
| Client secret | App registration **Certificates & secrets** | Shown once at creation |
| Redirect URI | App registration **Authentication** | `https://trove.nwks.com/auth/oidc/callback` |

### Trove Variables

| Variable | Required | Description |
|---|---|---|
| `TROVE_AUTH_MODE` | Yes | Set to `oidc` |
| `TROVE_PUBLIC_URL` | Yes | Public HTTPS URL for the Trove server |
| `TROVE_OIDC_ISSUER_URL` | Yes | Entra issuer URL: `https://login.microsoftonline.com/<tenant-id>/v2.0` |
| `TROVE_OIDC_CLIENT_ID` | Yes | Entra application client ID |
| `TROVE_OIDC_CLIENT_SECRET` | Yes | Entra client secret value |
| `TROVE_OIDC_REDIRECT_URL` | Yes | Trove callback URL registered in Entra |
| `TROVE_COOKIE_SECURE` | Recommended | Set to `true` for HTTPS browser sessions |

### Troubleshooting

| Symptom | Likely Cause | Fix |
|---|---|---|
| Entra shows `AADSTS50011` | Redirect URI mismatch | Make Entra's Web redirect URI exactly match `TROVE_OIDC_REDIRECT_URL` |
| Trove reports OIDC discovery failure | Wrong tenant ID, issuer URL, network egress, or proxy issue | Open the `.well-known/openid-configuration` URL from the Trove host |
| Token exchange fails | Wrong client secret or expired secret | Create a new client secret and update `TROVE_OIDC_CLIENT_SECRET` |
| Login succeeds but user name or email is empty | Entra did not return optional profile or email claims | Confirm Trove requests `openid profile email`; check the Entra user's email fields |
| Login loops or session is lost | Browser cookie rejected | Use HTTPS and set `TROVE_COOKIE_SECURE=true` |
| Users from another tenant cannot log in | App is single-tenant | Use the intended tenant or change the app registration account type |

### Security Notes

- Use HTTPS for every production Trove URL.
- Rotate Entra client secrets before expiration.
- Do not enable dev auth mode in production.
- Use Trove API tokens for agents and CI instead of browser sessions.
- Restrict network access to the Trove server and PostgreSQL according to your deployment model.

## Next

- Review [Authentication](/security/authentication) for all Trove auth modes
- Configure [API Tokens](/security/api-tokens) for CI and agents
- Review [RBAC & Scopes](/security/rbac-scopes) for write API access
