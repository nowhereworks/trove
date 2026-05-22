# Testing Strategy

## Commands

Current repo-wide command:

```bash
go test ./...
```

Current state: 192 tests across 17 packages.

Future focused examples once packages exist:

```bash
go test ./internal/manifest
go test ./internal/manifest -run TestValidateManifest
go test ./internal/api -run TestResolve
```

Do not add npm, Vite, lint, formatter, typecheck, or UI test commands until their manifests/configs exist.

## Test Tiers

Unit tests cover pure logic with no network or database dependency.

Handler tests cover HTTP behavior with in-process routers and fakes or test stores when possible.

PostgreSQL integration tests cover migrations, `sqlc` queries, constraints, triggers, and PostgreSQL-specific behavior.

CLI tests cover command output, exit codes, lockfile rewrites, install/update behavior, and JSON output.

UI smoke tests start after `web/` exists and cover the core screens and flows.

End-to-end tests can be added after the server, database, and CLI exist.

## PostgreSQL Integration Tests

Database integration tests must run against real PostgreSQL.

SQLite substitutes are not acceptable because PostgreSQL-specific constraints, triggers, arrays, JSONB, full-text search, and locking behavior matter.

Use this environment variable for integration tests once the harness exists:

```bash
TROVE_TEST_DATABASE_URL=postgres://trove:trove@localhost:5432/trove_test?sslmode=disable
```

Rules:

- Skip DB integration tests when `TROVE_TEST_DATABASE_URL` is unset.
- Run migrations against the test database before DB integration tests.
- Use isolated schemas/databases or clean up created data between tests.
- Test immutability at the database layer, not only through application services.
- Test full-text search against PostgreSQL, not mocked string matching.

## Fixture Layout

Use repo-local fixtures once implementation packages exist:

```text
internal/testfixtures/
  manifests/
    valid-trove.yaml
    invalid-missing-description.yaml
    invalid-path-traversal.yaml
    invalid-compatibility.yaml
  locks/
    valid.trove.lock.yaml
    stale.trove.lock.yaml
  packages/
    minimal/
      AGENTS.md
      trove.yaml
    optional-artifacts/
      AGENTS.md
      commands/review.md
      trove.yaml
```

Fixtures should be small, deterministic, and safe to publish in a public repository.

Do not store real tokens, private URLs, or customer/project names in fixtures.

## Required Unit Coverage

- Slug validation.
- Strict SemVer parsing.
- Selector parsing for exact versions, `latest`, `stable`, major, minor, and package digest selectors.
- Alias resolution ordering.
- Canonical package digest generation.
- Per-file digest generation.
- Manifest validation.
- Artifact path validation, including absolute paths and `..` traversal.
- Compatibility matching for tools, runtimes, and models.
- Strict compatibility behavior where `unknown` fails.
- Lockfile parsing.
- Canonical lockfile rewrite ordering.

## Required Handler Coverage

- `X-Request-Id` exists on success and error responses.
- Error bodies include the same request ID as the response header.
- JSON fields use camelCase.
- Cursor-paginated list responses use `items` and `nextCursor`.
- Resolve exact, `latest`, `stable`, major, minor, and digest selectors.
- Manifest endpoint returns the stored manifest for an exact version.
- Raw exact fetch returns artifact bytes, digest ETag, and immutable cache headers.
- Raw alias fetch redirects to the exact raw URL.
- Archive endpoints generate deterministic `.tar.gz` and `.zip` responses.
- Public resources allow anonymous reads for metadata, manifests, archives, raw artifacts, and resolve responses.
- Private and internal resources require auth.

## Required DB Integration Coverage

- All migrations apply cleanly to an empty PostgreSQL database.
- Down migrations work for local development when safe.
- `sqlc` query methods work against migrated schema.
- Slug constraints reject invalid org, namespace, and package slugs.
- Version constraints reject non-SemVer published versions.
- Visibility constraints accept only `private`, `internal`, and `public`.
- Published versions cannot mutate manifest, artifacts, version, or digest through direct SQL updates.
- Channel aliases can move between immutable versions.
- Full-text search returns published active packages by default.

## Required CLI Coverage

- Commands default to human-readable text output.
- Structured commands support `--json`.
- `resolve` prints exact version and digest.
- `fetch` can write to stdout or a target path.
- `install` installs required artifacts by default.
- Optional artifacts require explicit selection.
- `install` fails instead of overwriting a different existing target file unless overwrite is explicit.
- `check` exits zero for ordinary updates by default.
- `check` exits non-zero for yanked or incompatible installed versions.
- `check --fail-on-update` exits non-zero for ordinary updates.
- `update` is dry-run by default.
- `update --apply` writes changed files and rewrites `.trove.lock.yaml` as canonical YAML.

## Required UI Smoke Coverage

Add UI smoke tests only after `web/` and its test tooling exist.

Required smoke flows:

- Package search and browse.
- Package detail.
- Version detail.
- Raw URL copy controls.
- Upload validation error display.
- Submit for review.
- Approve or request changes with a simple comment.
- Publish after approval.

## CI Growth Path

Initial CI, once packages exist:

```bash
go test ./...
```

Add a PostgreSQL-backed integration job once migrations exist.

Add `sqlc` generation verification once `sqlc.yaml` exists.

Add UI build/test jobs only after `web/package.json` exists.

Do not require Docker Compose, npm, pnpm, yarn, lint, formatter, typecheck, migration, or codegen commands before their configs/manifests are committed.
