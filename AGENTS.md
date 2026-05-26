# AGENTS.md

## Workflow Rules **CRITICAL: MUST FOLLOW**

- Work in small, atomic units of work, committing between each unit.
- Execute new work on branches, then merge to `main` using the GitHub `gh` tool.
- Preserve durable user instructions or project facts in `AGENTS.md` or another discoverable Markdown file when asked to persist them.
- Keep `AGENTS.md` up to date, but make the smallest accurate edit instead of rewriting unrelated guidance.
- Whenever changing the Helm chart, bump the chart version so publishing does not fail.
- Before publishing or approval workflows are implemented, keep blocking checks in mind: manifest validation, route/version consistency, path validation, size limits, required metadata, secret scanning, and high-risk unsafe-instruction scanning.

## Current State

- Slices 1-5 (Thin Read Path, Draft Upload/Publish, Auth/Visibility/Review, Updates/CLI, Search And Adoption) are substantially implemented.
- `go test ./...` passes (191 tests across 19 packages).
- `go build ./...` succeeds.
- Runtime package storage is PostgreSQL-only; do not add an in-memory database/store fallback.
- Migrations: 000001_init, 000002_seed_sample, 000003_auth_review, 000004_search_projects.
- `sqlc` generated code is committed and covers read, write, and auth query paths.
- React/Vite/TypeScript SPA is implemented in `web/` and builds to `internal/ui/static/`.
- UI includes: package list, search, package detail, adoption dashboard, upload/publish flow, and review queue.
- RBAC scope checks are wired up on all write API routes.

## Commands

- Verify the Go toolchain with `go env GOVERSION`; this environment reports `go1.26.1`.
- `go test ./...` runs all Go tests.
- `cd web && npm run build` builds the React SPA to `internal/ui/static/`.
- DB integration tests must use real PostgreSQL and should be gated by `TROVE_TEST_DATABASE_URL` once a test harness exists; do not use SQLite substitutes.

## Brand Guidance

- `docs/trove_one_page_brand_kit_for_agents.md` is the source of truth for Trove's visual brand, UI styling direction, logo guidance, color palette, typography, iconography, and product copy tone.
- Apply the brand kit when changing frontend UI, docs visual presentation, landing pages, logos/icons, screenshots, and marketing-style copy.
- When brand direction is unclear, optimize for a secure developer-infrastructure product: dark, precise, durable, practical, self-hosted, technical, and easy to scan.

## Product Direction To Preserve

- Build Trove as a self-hosted Go single binary that serves JSON APIs, raw artifact endpoints, health/metrics, and an embedded React/Vite SPA.
- Use PostgreSQL as the MVP source of truth for metadata and artifact blobs; optional S3-compatible object storage can come later for large archives.
- Store MVP artifact bytes in PostgreSQL `bytea`; keep the design ready for near-future S3-compatible storage, preferably RustFS.
- Prefer `chi`, `pgx`, and `sqlc` if/when backend implementation starts.
- Use `sqlc` with `pgx` for SQL-first DB access.
- Use `golang-migrate/migrate` for SQL-first PostgreSQL migrations.
- Run migrations on startup only when explicitly enabled for local development/tests; production should use an explicit migration command/job.
- Use UUID v7 primary keys and `timestamptz` UTC timestamps.
- Prefer React + Vite + TypeScript with React Router and Radix UI for the UI static output; avoid a Next.js runtime unless the single-binary deployment goal changes.
- Defer Git-backed storage, OpenSearch, OCI compatibility, advanced policy engines, and full provenance/signing unless explicitly implementing later phases.
- Use PostgreSQL full-text search for MVP package discovery; default search should show published active packages only.
- MVP uses exactly one namespace level: `org/namespace/package`.
- Package names are unique within a namespace; manifests and lockfiles must use full `org/namespace/package` references.
- Org, namespace, and package slugs use `^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`; slug renames are deferred for MVP.
- Raw artifact URLs require auth by default, but namespaces/packages can be explicitly public like public GitHub repos.
- Visibility values are `private`, `internal`, and `public`; effective visibility is the most restrictive value across org, namespace, package, and version.
- Public anonymous reads include package metadata, manifests, archives, raw artifacts, and resolve responses only.
- Use OIDC for production human auth, scoped opaque bearer tokens for agents/CI, and gated dev auth only for local development/tests.
- Keep OIDC standards-based and provider-neutral; configure provider details rather than baking in one provider.
- Include an MVP CLI prototype for resolve, fetch, install, check, and dry-run update flows.
- CLI defaults to human-readable text output and should support `--json` for agents/CI.
- `trove check` is advisory for ordinary updates by default; yanked or incompatible installed versions should fail unless policy says otherwise.
- `trove install` installs required artifacts by default; optional artifacts require explicit selection.
- `trove update` is dry-run by default and requires explicit apply to write files or lockfiles.
- MVP web content management is upload-only; browser-based artifact editing is deferred.
- MVP publish flow requires one human approval; maintainers publish after approval; content changes after submission reset approvals/checks.
- MVP compatibility uses a generic tools/models/runtimes schema; tool versions use SemVer ranges, runtimes are named strings, and model constraints use family plus optional context window.
- Unknown compatibility is advisory by default but fails in strict compatibility mode.

## Domain Rules

- Published package versions are immutable; fixes require a new version, not mutation.
- Enforce published-version immutability with app checks plus PostgreSQL triggers/constraints.
- Published versions use strict SemVer `MAJOR.MINOR.PATCH`; prerelease and arbitrary string versions are deferred.
- Aliases `latest`, `stable`, `v3`, and `v3.2` resolve server-side to exact immutable versions; MVP named aliases are only `latest` and `stable`.
- Alias raw URLs redirect to exact raw URLs; yanked versions remain fetchable by exact version or package digest for authorized clients.
- Agent-facing APIs matter as much as the web UI: raw file fetch, manifest fetch, resolve, archive download, update check, and compatibility checks are first-class.
- API JSON uses camelCase; list endpoints use cursor pagination; every response should include `X-Request-Id`.
- Keep registry-style route refs like `/api/v1/resolve/{org}/{namespace}/{package}@{selector}`.
- Package manifests are `trove.yaml`; package installs may be pinned by `.trove.lock.yaml`.
- Published manifests require non-empty descriptions and at least one maintainer; manifest dependencies are declare-only in MVP.
- Artifact `targetPath` defaults to `path`; installs must not overwrite different existing files without an explicit overwrite option.
- MVP package uploads support `.tar.gz` and `.zip`; archive downloads are generated on demand from stored artifact files.
- Package version digests cover canonical manifest JSON plus sorted artifact paths, types, target paths, file digests, and sizes.
- Registry changes should usually be surfaced through CLI, CI, Backstage, or agent proposals; the registry should not directly edit downstream repos by default.
- Adoption reporting comes from configured CLI install/check/update flows; public dashboards show aggregate counts by default.

## Documentation

User-facing docs live in `docs-site/` (VitePress static site). Keep docs synchronized with implementation.

### Rule: Feature Change → Docs Update

When adding, modifying, or removing a feature, update the matching docs page:

| Feature area | Docs page(s) to update |
|---|---|
| New API endpoint | `api/public-apis.md` or `api/management-apis.md` |
| API response shape change | Matching page under `api/` |
| New CLI command or flag | Matching page under `cli/` |
| CLI output format change | Matching page under `cli/` |
| New config option | `operations/configuration.md` |
| New artifact type | `concepts/artifact-types.md` |
| New lifecycle state | `concepts/lifecycle-states.md` |
| New visibility rule | `concepts/visibility.md` |
| New auth mode or scope | `security/authentication.md` or `security/rbac-scopes.md` |
| New review/approval behavior | `publishing/review-workflow.md` |
| New security scan pattern | `publishing/security-scanning.md` |
| New manifest field or validation rule | `publishing/manifests.md` |
| New search filter or behavior | `discovery/search.md` |
| New adoption metric | `discovery/adoption-dashboard.md` |
| New migration or schema change | `operations/deployment.md` |
| New compatibility rule | `operations/compatibility.md` |

### Rule: New Feature → New Docs Page

When implementing a feature that has no matching docs page:

1. Create the page under the appropriate `docs-site/` subdirectory
2. Follow the Why → How → Reference → Next pattern
3. Use `nwks` org in all examples (e.g., `nwks/platform/agent-backend@stable`)
4. Add the page to the sidebar in `.vitepress/config.ts`
5. Link to it from the landing page `docs-site/index.md`
6. Add a row to the feature-to-page mapping table above

### Documentation Standards

- Tutorial-first: explain the problem (Why), then step-by-step usage (How), then reference tables
- Use `nwks` (Nowhereworks) as the fictional org in all examples
- Document only what is implemented; do not document deferred features
- Keep examples copy-paste runnable where possible
- Reference tables should include: fields, types, required/optional, defaults, and descriptions
