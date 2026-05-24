# Trove Implementation Specification

Trove is a curated registry for agent-consumable engineering artifacts such as `AGENTS.md`, `SKILL.md`, command definitions, reusable prompts, context packs, templates, model routing policies, review rubrics, and project scaffolding instructions.

The monolithic product spec has been split into focused implementation documents under `docs/spec/`. Treat those files as the canonical source of truth.

## Reading Order

1. [`spec/00-overview.md`](spec/00-overview.md) defines the product, goals, non-goals, personas, MVP success criteria, and deferred scope.
2. [`spec/01-domain-model.md`](spec/01-domain-model.md) defines organizations, namespaces, packages, versions, artifact types, selectors, lifecycle, visibility, and immutability.
3. [`spec/02-manifest-and-lockfile.md`](spec/02-manifest-and-lockfile.md) defines `trove.yaml`, `.trove.lock.yaml`, validation rules, and path rules.
4. [`spec/03-api-contract.md`](spec/03-api-contract.md) defines agent APIs, management APIs, raw file behavior, update checks, auth expectations, and error responses.
5. [`spec/04-data-model.md`](spec/04-data-model.md) defines the PostgreSQL MVP schema, constraints, and indexes.
6. [`spec/05-security-governance.md`](spec/05-security-governance.md) defines authentication, RBAC, visibility, review workflow, scanning, and audit events.
7. [`spec/06-ui-requirements.md`](spec/06-ui-requirements.md) defines the React/Vite MVP UI, upload-only artifact management, and review screens.
8. [`spec/07-cli-and-integrations.md`](spec/07-cli-and-integrations.md) defines the MVP CLI prototype and OpenCode, CI, Backstage, and Git provider integrations.
9. [`spec/08-ops-and-config.md`](spec/08-ops-and-config.md) defines deployment shape, stack choices, config, health, metrics, and caching.
10. [`spec/09-implementation-plan.md`](spec/09-implementation-plan.md) defines the target repo structure, vertical slices, and acceptance criteria.
11. [`spec/10-decisions.md`](spec/10-decisions.md) records accepted decisions, deferred decisions, and remaining open questions.
12. [`spec/11-testing.md`](spec/11-testing.md) defines test tiers, fixtures, commands, integration prerequisites, and required coverage.
13. [`spec/12-command-semantics.md`](spec/12-command-semantics.md) defines Trove-wide package references, selectors, raw URLs, and CLI command families.

## Key MVP Decisions

- Build a self-hosted Go single binary with embedded React/Vite static UI.
- Use PostgreSQL as the MVP source of truth for metadata and artifact blobs.
- Use exactly one namespace level in MVP: `org/namespace/package`.
- Require authentication for raw artifacts by default, while allowing explicitly public namespaces and packages.
- Include an MVP CLI prototype for resolve, download/fetch, install, check, dry-run update, and editable worktree flows.
- Keep MVP web content management upload-only; browser-based artifact editing is deferred.
- Published package versions are immutable; fixes require a new version.
- Aliases such as `latest`, `stable`, `v3`, and `v3.2` resolve server-side to exact immutable versions.
- Agent-facing APIs are first-class and must not depend on the web UI.
- Defer Git-backed storage, OpenSearch, OCI compatibility, advanced policy engines, and full signing/provenance.

## Implementation Entry Point

Start with [`spec/09-implementation-plan.md`](spec/09-implementation-plan.md), Slice 1.
