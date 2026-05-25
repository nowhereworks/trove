# find-trove-skills Implementation Plan

## Scope

Implement `find-trove-skills` as a bundled Trove core skill, expose it without RBAC, and add `trove skills find` as the Trove-native replacement for `npx skills find`.

## Files To Add

- `internal/core/skills/skills/find-trove-skills/SKILL.md`
- `internal/core/skills/skills.go`
- `internal/cli/skills.go`
- `docs-site/cli/skills.md`

## Files To Modify

- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/cli/cli.go`
- `internal/cli/client.go`
- `internal/cli/client_test.go`
- `internal/cli/cli_test.go`
- `docs-site/.vitepress/config.ts`
- `docs-site/index.md`
- `docs-site/api/public-apis.md`

## Bundled Skill

- Store the adapted source at `internal/core/skills/skills/find-trove-skills/SKILL.md`.
- Use Go `embed.FS` so it ships inside the Trove binary.
- Use this frontmatter:

```yaml
---
name: find-trove-skills
description: Helps users discover and install Trove-hosted agent skills when they ask for reusable agent capabilities.
---
```

- Adapt upstream `find-skills` content from `https://github.com/vercel-labs/skills/blob/main/skills/find-skills/SKILL.md` with these replacements:
- `find-skills` -> `find-trove-skills`
- `npx skills find [query]` -> `trove skills find [query]`
- `npx skills add ...` -> `trove install <org/namespace/package@selector> --optional`
- `npx skills check` -> `trove check`
- `npx skills update` -> `trove update`
- `skills.sh` URLs -> Trove web/API URLs, such as `/search?artifactType=skill`, `/packages/{org}/{namespace}/{package}`, and `/api/v1/search/packages?q=...&artifactType=skill`
- Include this attribution line in the body:

```text
Adapted from https://github.com/vercel-labs/skills/blob/main/skills/find-skills/SKILL.md
```

- Ensure bundled content contains no `npx skills` or `skills.sh` references.

## Core Skill Package

- `internal/core/skills/skills.go` should expose `Read(name string) ([]byte, bool)`.
- Support exactly `find-trove-skills` for this change.
- Unknown names return `false`, which the API maps to 404.

## Unauthenticated API Route

- Add exact route: `GET /api/v1/core/skills/find-trove-skills/SKILL.md`.
- Do not wrap the route in `auth.RequireAuth` or scope checks.
- Return `Content-Type: text/markdown; charset=utf-8`.
- Keep normal `X-Request-Id` middleware behavior.
- Return 200 with embedded skill content.
- This must work with or without PostgreSQL because it does not use package storage.

## CLI Command

- Add namespace command: `trove skills find [query] [--json]`.
- `Run` handles top-level `skills`.
- `IsCLICommand` returns true for `skills`.
- `printUsage` includes `skills find`.
- Query args are joined with spaces, so `trove skills find react performance` searches `react performance`.
- If query is empty, default query to `skill`.
- Search API call uses `artifactType=skill`.
- Human output should be compact:

```text
Skills matching "react performance":
- nwks/platform/react-best-practices@stable
  React and Next.js performance optimization guidelines.
```

- Empty result output:

```text
No Trove skills found for "react performance".
```

- JSON output returns the API search response shape.

## Client Support

- Add CLI client types:
- `SearchPackagesParams`
- `SearchPackagesResponse`
- `PackageSummary`
- Add `func (c *Client) SearchPackages(params SearchPackagesParams) (*SearchPackagesResponse, error)`.
- Build the request URL with `url.Values`.
- Include `q` and `artifactType`; cursor and limit can be added only if needed.

## Tests

- API test: `GET /api/v1/core/skills/find-trove-skills/SKILL.md` returns 200.
- API test: response includes `find-trove-skills`.
- API test: response includes `trove skills find`.
- API test: response does not include `npx skills`.
- API test: no auth header is required.
- CLI routing test: `IsCLICommand([]string{"skills"}) == true`.
- CLI routing test: `trove skills bogus` returns unknown skills subcommand.
- Client test: `SearchPackages` calls `/api/v1/search/packages?q=react+performance&artifactType=skill`.
- Client test: `SearchPackages` decodes `items` and `nextCursor`.
- CLI command test: mock HTTP server returns one skill package and `Run([]string{"skills", "find", "react", "performance"})` prints the package ref.
- CLI command test: `Run([]string{"skills", "find", "react", "--json"})` prints JSON.

## Docs

- Add `docs-site/cli/skills.md` using the Why -> How -> Reference -> Next structure.
- Add `skills` to the VitePress CLI sidebar.
- Add a `skills` card to the docs landing page CLI section.
- Update `docs-site/api/public-apis.md` endpoint table with:

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/core/skills/find-trove-skills/SKILL.md` | none | Fetch bundled find-trove-skills source |

- Document `trove skills find [query]` examples using `nwks`.

## Verification

- Run `go test ./...`.
- Optional when docs dependencies are available: `cd docs-site && npm run docs:build`.
