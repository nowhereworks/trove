# trove skills

## Why

Agent skills are reusable operating instructions, workflows, and tools for agents. `trove skills find` searches your Trove registry for packages that contain `skill` artifacts, so teams can discover approved internal skills instead of copying instructions between repositories.

## How

### Find Skills

Search by keyword:

```bash
trove skills find react performance
```

Output:

```text
Skills matching "react performance":
- nwks/platform/react-best-practices@latest
  React and Next.js performance optimization guidelines.
```

If you omit the query, Trove searches for `skill`:

```bash
trove skills find
```

### Install a Skill Package

Install the selected package with optional artifacts included:

```bash
trove install nwks/platform/react-best-practices@latest --optional
```

Use `--optional` because skill artifacts can be optional in packages that also include AGENTS.md files, prompts, commands, or templates.

### JSON Output

Use `--json` when another tool or agent needs the raw search response:

```bash
trove skills find react performance --json
```

Output:

```json
{
  "items": [
    {
      "org": "nwks",
      "namespace": "platform",
      "name": "react-best-practices",
      "displayName": "React Best Practices",
      "description": "React and Next.js performance optimization guidelines.",
      "visibility": "public",
      "lifecycle": "active",
      "latestVersion": "1.0.0"
    }
  ],
  "nextCursor": null
}
```

### No Results

When no matching packages are found, the command prints:

```text
No Trove skills found for "react performance".
```

Try a broader query, a synonym, or the web search page with the `skill` artifact type filter.

## Reference

### Command

```bash
trove skills find [query] [--json]
```

| Argument or flag | Type | Required | Default | Description |
|---|---|---|---|---|
| `query` | string | no | `skill` | Search terms. Multiple words are joined with spaces. |
| `--json` | boolean | no | `false` | Print the API search response shape as JSON. |

### API Used

`trove skills find` calls:

```text
GET /api/v1/search/packages?q={query}&artifactType=skill
```

The response is the same paginated package search shape documented in [Public APIs](/api/public-apis).

### Bundled Finder Skill

Trove also ships a bundled core skill that teaches agents how to search for Trove-hosted skills:

```bash
trove download core/skills/find-trove-skills/SKILL.md --output SKILL.md
```

The underlying unauthenticated API endpoint is:

```text
GET /api/v1/core/skills/find-trove-skills/SKILL.md
```

This endpoint does not require authentication and does not require package storage.

## Next

- [Search](/discovery/search) - learn the package discovery filters
- [install](/cli/install) - install selected skill artifacts
- [Public APIs](/api/public-apis) - call search and core skill endpoints directly
