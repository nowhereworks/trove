# Search

## Why

When your organization has dozens of packages across multiple namespaces, finding the right one matters. You might need "a skill for API review," "all Go-related agent instructions," or "packages compatible with Claude." Search lets you discover packages by name, description, type, tool compatibility, and more.

## How

### Basic Search

```bash
GET /api/v1/search/packages?q=backend+agent
```

Response:

```json
{
  "items": [
    {
      "org": "nwks",
      "namespace": "platform",
      "name": "agent-backend",
      "displayName": "Backend Agent Defaults",
      "description": "Default agent instructions, skills, and commands for backend services.",
      "latestVersion": "1.0.0",
      "visibility": "public",
      "lifecycle": "active",
      "labels": { "language": "golang", "framework": "chi" },
      "artifactTypes": ["agent-instructions", "skill", "command"],
      "tools": ["opencode"]
    }
  ],
  "nextCursor": null
}
```

### Default Behavior

- Returns **published active packages only**
- Draft, review, yanked, and archived versions are excluded
- Deprecated packages are included only when explicitly filtered and authorized
- Results are ranked by relevance using PostgreSQL full-text search

### Filters

| Filter | Query Param | Example |
|---|---|---|
| Free-text query | `q` | `?q=backend+agent` |
| Organization | `org` | `?org=nwks` |
| Namespace | `namespace` | `?namespace=platform` |
| Artifact type | `artifactType` | `?artifactType=skill` |
| Language label | `language` | `?language=golang` |
| Framework label | `framework` | `?framework=chi` |
| Tool compatibility | `tool` | `?tool=opencode` |
| Visibility | `visibility` | `?visibility=public` |
| Lifecycle | `lifecycle` | `?lifecycle=active` |

### Combining Filters

```bash
# Find Go skills compatible with opencode in the platform namespace
GET /api/v1/search/packages?q=skill&namespace=platform&artifactType=skill&language=golang&tool=opencode
```

### Pagination

Search uses cursor pagination:

```bash
# First page
GET /api/v1/search/packages?q=agent&limit=20

# Next page (use cursor from previous response)
GET /api/v1/search/packages?q=agent&limit=20&cursor=eyJpZCI6IjAxOTA...
```

| Param | Type | Default | Description |
|---|---|---|---|
| `limit` | integer | 50 | Number of results per page |
| `cursor` | string | — | Opaque cursor from previous response |

### What is Indexed

Search indexes the following package metadata:

- Organization, namespace, and package slugs
- Display name and description
- Labels (language, framework, maturity, etc.)
- Artifact paths and types
- Compatibility tools, runtimes, and model families
- Maintainer names and teams

Artifact file **contents** are not indexed in the MVP.

### Search UI

The Trove UI provides a search page with:

- Text input for free-text queries
- Filter dropdowns for org, namespace, artifact type, tool, visibility, and lifecycle
- Result cards showing package name, description, latest version, and labels
- Pagination controls

### Next Steps

- See the [Adoption Dashboard](/discovery/adoption-dashboard) for usage statistics
- Learn about [Public APIs](/api/public-apis) for programmatic search
