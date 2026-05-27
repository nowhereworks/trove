# Visibility

## Why

Not all packages should be readable by everyone. Some are private to a team, and some are public like open-source packages. Trove needs a visibility model that controls anonymous read access.

## How

### Visibility Levels

| Level | Meaning |
|---|---|
| `private` | Requires authenticated user or token |
| `internal` | Requires authentication within the owning installation |
| `public` | Metadata, manifests, archives, and raw files are anonymously readable |

### Package-Level Visibility

Visibility is set at the **package level** via the management API. All versions inherit the package visibility — there is no per-version or per-namespace visibility.

```
PATCH /api/v1/packages/{org}/{namespace}/{package}/visibility
{ "visibility": "public" }
```

Requires `package:write` scope.

### Anonymous Read Rules

When a package is `public`, anonymous users can read:

- Package metadata (name, description, versions list)
- Manifests (`Trovefile`)
- Archives (`.tar.gz`, `.zip`)
- Raw artifact files
- Resolve responses

Anonymous users **cannot** read:

- Audit events
- Review details and comments
- Draft versions
- Non-public packages
- Token metadata
- User/team membership
- Administrative settings

### Default Visibility

New packages default to `private` unless explicitly set otherwise during creation.

### Configuration

Raw artifact URLs require authentication by default. Public packages must be explicitly enabled:

```yaml
raw:
  requireAuthByDefault: true
  allowPublicPackages: true
```

### Example: Making a Package Public

```bash
# Create a package (defaults to private)
POST /api/v1/packages
{
  "org": "nwks",
  "namespace": "platform",
  "name": "react-defaults",
  "displayName": "React Defaults"
}

# Make it public
PATCH /api/v1/packages/nwks/platform/react-defaults/visibility
{ "visibility": "public" }

# Anyone can now read the package anonymously:
GET /api/v1/packages/nwks/platform/react-defaults
GET /raw/nwks/platform/react-defaults/AGENTS.md@1.0.0
```

### Next Steps

- Learn how [Lifecycle States](/concepts/lifecycle-states) affect package access
- Understand [Authentication](/security/authentication) for accessing private resources
