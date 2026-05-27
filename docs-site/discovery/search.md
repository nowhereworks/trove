# Search

## Why

When your organization has dozens of packages across multiple namespaces, finding the right one matters. You might need "a skill for API review," "all Go-related agent instructions," or "packages for Claude." Search lets you discover packages by name, description, type, and more.

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
      "lifecycle": "active"
    }
  ],
  "nextCursor": null
}
```

### Default Behavior

- Returns **published active packages only**
- Draft, review, and yanked versions are excluded from search results
- Results are currently limited to public active packages
- Results are ranked by relevance using PostgreSQL full-text search

### Filters

| Filter | Query Param | Example |
|---|---|---|
| Free-text query | `q` | `?q=backend+agent` |
| Organization | `org` | `?org=nwks` |
| Namespace | `namespace` | `?namespace=platform` |
| Artifact type | `artifactType` | `?artifactType=skill` |

### Combining Filters

```bash
# Find Go skills in the platform namespace
GET /api/v1/search/packages?q=skill&namespace=platform&artifactType=skill
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
- Artifact paths and types

Artifact file **contents** are not indexed in the MVP.

### Search UI

The Trove UI provides a search page with:

- Text input for free-text queries
- Result cards showing package name, description, and visibility
- Pagination controls

### Next Steps

- See the [Adoption Dashboard](/discovery/adoption-dashboard) for usage statistics
- Learn about [Public APIs](/api/public-apis) for programmatic search
