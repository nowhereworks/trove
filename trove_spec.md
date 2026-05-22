# Trove Implementation Specification

Trove is a curated registry for agent-consumable engineering artifacts such as `AGENTS.md`, `SKILL.md`, command definitions, reusable prompts, context packs, templates, model routing policies, review rubrics, and project scaffolding instructions.

The monolithic product spec has been split into focused implementation documents under `docs/spec/`. Treat those files as the canonical source of truth.

## Reading Order

1. [`docs/spec/00-overview.md`](docs/spec/00-overview.md) defines the product, goals, non-goals, personas, MVP success criteria, and deferred scope.
2. [`docs/spec/01-domain-model.md`](docs/spec/01-domain-model.md) defines organizations, namespaces, packages, versions, artifact types, selectors, lifecycle, visibility, and immutability.
3. [`docs/spec/02-manifest-and-lockfile.md`](docs/spec/02-manifest-and-lockfile.md) defines `trove.yaml`, `.trove.lock.yaml`, validation rules, and path rules.
4. [`docs/spec/03-api-contract.md`](docs/spec/03-api-contract.md) defines agent APIs, management APIs, raw file behavior, update checks, auth expectations, and error responses.
5. [`docs/spec/04-data-model.md`](docs/spec/04-data-model.md) defines the PostgreSQL MVP schema, constraints, and indexes.
6. [`docs/spec/05-security-governance.md`](docs/spec/05-security-governance.md) defines authentication, RBAC, visibility, review workflow, scanning, and audit events.
7. [`docs/spec/06-ui-requirements.md`](docs/spec/06-ui-requirements.md) defines the React/Vite MVP UI, upload-only artifact management, and review screens.
8. [`docs/spec/07-cli-and-integrations.md`](docs/spec/07-cli-and-integrations.md) defines the MVP CLI prototype and OpenCode, CI, Backstage, and Git provider integrations.
9. [`docs/spec/08-ops-and-config.md`](docs/spec/08-ops-and-config.md) defines deployment shape, stack choices, config, health, metrics, and caching.
10. [`docs/spec/09-implementation-plan.md`](docs/spec/09-implementation-plan.md) defines the target repo structure, vertical slices, and acceptance criteria.
11. [`docs/spec/10-decisions.md`](docs/spec/10-decisions.md) records accepted decisions, deferred decisions, and remaining open questions.
12. [`docs/spec/11-testing.md`](docs/spec/11-testing.md) defines test tiers, fixtures, commands, integration prerequisites, and required coverage.

## Key MVP Decisions

- Build a self-hosted Go single binary with embedded React/Vite static UI.
- Use PostgreSQL as the MVP source of truth for metadata and artifact blobs.
- Use exactly one namespace level in MVP: `org/namespace/package`.
- Require authentication for raw artifacts by default, while allowing explicitly public namespaces and packages.
- Include an MVP CLI prototype for resolve, fetch, install, check, and dry-run update flows.
- Keep MVP web content management upload-only; browser-based artifact editing is deferred.
- Published package versions are immutable; fixes require a new version.
- Aliases such as `latest`, `stable`, `v3`, and `v3.2` resolve server-side to exact immutable versions.
- Agent-facing APIs are first-class and must not depend on the web UI.
- Defer Git-backed storage, OpenSearch, OCI compatibility, advanced policy engines, and full signing/provenance.

## Implementation Entry Point

Start with [`docs/spec/09-implementation-plan.md`](docs/spec/09-implementation-plan.md), Slice 1.
