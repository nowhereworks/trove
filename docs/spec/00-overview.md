# Overview

## Product

AgentHub is a curated registry for agent-consumable engineering artifacts:

- `AGENTS.md`
- `SKILL.md`
- command definitions
- reusable prompts
- context packs
- templates
- model routing policies
- review rubrics
- project scaffolding instructions

It behaves like a lightweight GitHub/npm/OCI-style registry for agent instructions.

## Primary Goals

- Humans can discover, review, version, approve, and maintain artifacts through a web UI.
- Agents can fetch immutable raw artifacts over HTTP.
- Projects can pin exact versions, subscribe to update channels, and check for newer compatible releases.
- Organizations can define blessed defaults, ownership, approval workflows, security policy, governance rules, and lifecycle states.

## Non-Goals

- Do not build a full source-code Git hosting platform.
- Do not build a general-purpose package manager for binary artifacts.
- Do not replace Backstage or an internal developer portal.
- Do not build a full CI/CD system.
- Do not make the UI more important than agent-facing APIs.

## Personas

- Platform engineers publish and maintain blessed artifact packages.
- Application developers adopt packages and track updates.
- Agents and CLIs resolve versions, fetch raw files, and check compatibility.
- Security and architecture reviewers approve packages and enforce policy.
- Team maintainers own domain-specific packages.

## MVP Success Criteria

- A maintainer can create and publish `companyx/platform/agent-backend@1.0.0`.
- A human can browse the package and version in the UI.
- An agent can fetch `AGENTS.md` over an immutable raw URL.
- `@stable` resolves to an exact immutable version.
- A project can store `.agenthub.lock.yaml`.
- The update API can report that a newer compatible version exists.
- Published artifact content cannot be mutated.
- Publish actions are audited.
- Private artifacts require auth; public artifacts are explicitly marked public.
- The system runs as a Go binary plus PostgreSQL.

## Deferred Capabilities

- Git-backed storage.
- OpenSearch or advanced semantic search.
- OCI registry compatibility.
- Advanced policy engine.
- Full artifact signing and provenance.
- Multi-region replication.
- Editor plugins.
