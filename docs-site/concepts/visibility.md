# Visibility

## Why

Not all packages should be readable by everyone. Some are private to a team, some are internal to the organization, and some are public like open-source packages. Trove needs a visibility model that works like GitHub repos — private repos inside public orgs, public repos inside private orgs — with clear inheritance rules.

## How

### Visibility Levels

| Level | Meaning |
|---|---|
| `private` | Requires authenticated user or token with explicit access |
| `internal` | Requires authentication within the owning installation |
| `public` | Metadata, manifests, archives, and raw files are anonymously readable |

### Most-Restrictive Inheritance

Visibility can be set at four levels: organization, namespace, package, and version. The **effective visibility** is the most restrictive value across all layers:

```
private > internal > public
```

Examples:

| Org | Namespace | Package | Version | Effective |
|---|---|---|---|---|
| `public` | `public` | `public` | — | `public` — anonymously readable |
| `public` | `public` | `private` | — | `private` — requires auth |
| `private` | `public` | `public` | — | `private` — org is most restrictive |
| `public` | `internal` | `public` | — | `internal` — requires installation auth |

### Anonymous Read Rules

When a resource is effectively `public`, anonymous users can read:

- Package metadata (name, description, versions list)
- Manifests (`trove.yaml`)
- Archives (`.tar.gz`, `.zip`)
- Raw artifact files
- Resolve responses

Anonymous users **cannot** read:

- Audit events
- Review details and comments
- Draft versions
- Non-public versions
- Token metadata
- User/team membership
- Administrative settings

### Default Visibility

The default visibility is `private` unless an organization or namespace policy specifies otherwise.

### Configuration

Raw artifact URLs require authentication by default. Public namespaces and packages must be explicitly enabled:

```yaml
raw:
  requireAuthByDefault: true
  allowPublicNamespaces: true
  allowPublicPackages: true
```

### Example: Making a Package Public

```bash
# Create a public namespace
POST /api/v1/orgs/nwks/namespaces
{ "slug": "open-source", "displayName": "Open Source", "visibility": "public" }

# Create a public package in that namespace
POST /api/v1/packages
{
  "org": "nwks",
  "namespace": "open-source",
  "name": "react-defaults",
  "displayName": "React Defaults",
  "visibility": "public"
}

# Anyone can now read the package anonymously:
GET /api/v1/packages/nwks/open-source/react-defaults
GET /raw/nwks/open-source/react-defaults/AGENTS.md@1.0.0
```

### Next Steps

- Learn how [Lifecycle States](/concepts/lifecycle-states) affect visibility
- Understand [Authentication](/security/authentication) for accessing private resources
