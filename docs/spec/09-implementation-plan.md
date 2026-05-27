# Implementation Plan

## Target Repo Structure

```text
trove/
  cmd/trove/main.go
  internal/api/
  internal/auth/
  internal/config/
  internal/db/
    queries/
  internal/manifest/
  internal/packages/
  internal/raw/
  internal/reviews/
  internal/search/
  internal/security/
  internal/updates/
  internal/ui/
  migrations/
  sqlc.yaml
  docs/
    trove_spec.md
    spec/
  web/
    package.json
    vite.config.ts
    src/
  examples/
```

## Slice 1: Thin Read Path

Goal: prove the single-binary registry shape.

1. Start Go HTTP server.
2. Add config loading.
3. Add PostgreSQL connection.
4. Add `golang-migrate/migrate` SQL migrations for org, namespace, package, version, artifact blob, artifact location, alias, and audit tables.
5. Seed one sample published package.
6. Implement resolve endpoint.
7. Implement manifest endpoint.
8. Implement exact raw file endpoint.
9. Serve embedded placeholder SPA.
10. Add package list and package detail UI backed by APIs.

Acceptance:

- `go test ./...` passes.
- `sqlc` generated query code is committed once `sqlc.yaml` exists.
- A seeded package resolves from `@latest` to an exact version.
- A raw exact URL returns `AGENTS.md` with digest ETag.
- API responses include `X-Request-Id`.
- JSON responses use camelCase fields.
- The UI can show the seeded package.

## Slice 2: Draft Upload And Publish

Goal: make package publishing real.

1. Add draft version creation.
2. Add artifact upload API.
3. Add `Trovefile` parsing and validation.
4. Add path validation.
5. Add digest computation.
6. Add publish endpoint.
7. Enforce immutability for published versions.
8. Add audit events for publish actions.
9. Add UI upload and publish flow.

Acceptance:

- A maintainer can upload a package draft.
- Invalid manifests are rejected with actionable errors.
- Published content cannot be mutated.
- Publishing writes audit events.

## Slice 3: Auth, Visibility, And Review

Goal: make the registry enterprise-safe.

1. Add OIDC browser login.
2. Add API tokens.
3. Add RBAC checks.
4. Add namespace and package visibility.
5. Add public namespace/package read behavior.
6. Add submit/review/approve workflow.
7. Add secret scanning and unsafe instruction scanning hooks.

Acceptance:

- Private raw URLs require auth.
- Public package raw URLs can be read anonymously.
- Self-approval is blocked by default.
- Publishing requires configured approvals.

## Slice 4: Updates And CLI Prototype

Goal: make installed packages maintainable.

1. Add `.trove.lock.yaml` parser.
2. Add update check API.
3. Add compatibility checks.
4. Add `trove resolve`.
5. Add `trove download` as the only single-artifact retrieval command; do not support `trove fetch`.
6. Add `trove install`.
7. Add `trove check`.
8. Add dry-run-by-default `trove update` and explicit `trove update --apply`.

Acceptance:

- CLI can install a package and write a lock file.
- CLI can report a newer compatible version.
- CLI can warn for yanked or incompatible installed versions.

## Slice 5: Search And Adoption

Goal: improve discovery and governance visibility.

1. Add PostgreSQL full-text search.
2. Add search UI.
3. Add project registration API.
4. Add project artifact install tracking.
5. Add adoption dashboard.

Acceptance:

- Users can search by name, description, labels, language, framework, tool compatibility, and artifact type.
- Default search returns published active packages only.
- Platform engineers can see projects using a package version.
- Public package adoption views show aggregate counts unless the user has detailed access.

## Initial Test Strategy

Use [`docs/spec/11-testing.md`](11-testing.md) as the canonical testing strategy.

Minimum implementation expectations:

- Add unit tests with each pure package.
- Add handler tests with each HTTP surface.
- Add PostgreSQL integration tests when migrations, `sqlc` queries, constraints, or triggers are introduced.
- Add CLI tests when CLI commands are introduced.
- Add UI smoke tests only after `web/` and its test tooling exist.
