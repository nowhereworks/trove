# What is Trove?

## Why

Agent tooling has a discovery and governance problem. Teams write `AGENTS.md` files, reusable skills, command definitions, prompts, and review rubrics — but there is no central place to publish, version, review, and distribute them. Files get duplicated across repos, drift out of sync, and no one knows who is using which version.

Trove solves this by acting as a curated registry for agent-consumable artifacts. It behaves like a lightweight npm/OCI-style registry, but for instruction files rather than binary packages.

## How

### The Hierarchy

Every package in Trove lives in a fixed hierarchy:

```
Organization → Namespace → Package → Version → Artifacts
```

For example:

```
nwks/platform/agent-backend@1.0.0
```

- **Organization** (`nwks`) — the owning entity, like a company or team group
- **Namespace** (`platform`) — a collection within the org, like a team or domain
- **Package** (`agent-backend`) — a named set of versioned artifacts
- **Version** (`1.0.0`) — an immutable release with a manifest and files
- **Artifacts** — the actual files: `AGENTS.md`, skills, commands, prompts, etc.

### What Trove Provides

| Capability | Description |
|---|---|
| **Publishing** | Upload `.tar.gz` or `.zip` archives, validate manifests, submit for review, and publish immutable versions |
| **Discovery** | Search packages by name, description, artifact type, tool compatibility, and more |
| **Agent APIs** | Resolve selectors to exact versions, fetch raw artifacts, download archives |
| **CLI** | Install, check for updates, and manage lockfiles from the command line |
| **Governance** | RBAC, review workflows, approval gates, security scanning, and audit trails |
| **Adoption Tracking** | See which projects use your packages through CLI-reported install data |

### Who Uses Trove

- **Platform engineers** publish and maintain blessed artifact packages
- **Application developers** discover and adopt packages for their projects
- **Agents and CLIs** resolve versions, fetch raw files, and check compatibility
- **Security reviewers** approve packages and enforce policy before publishing

### Architecture

Trove runs as a single Go binary backed by PostgreSQL:

```
┌─────────────────────────┐
│     Go Single Binary     │
│  ┌────────────────────┐  │
│  │   JSON APIs         │  │
│  │   Raw Endpoints     │  │
│  │   Embedded SPA UI   │  │
│  │   Health / Metrics  │  │
│  └────────────────────┘  │
└──────────┬───────────────┘
           │
┌──────────▼───────────────┐
│      PostgreSQL           │
│  Metadata + Blob Storage  │
└──────────────────────────┘
```

The binary serves JSON APIs, raw artifact endpoints, an embedded React SPA, and health/metrics endpoints. All metadata and artifact blobs are stored in PostgreSQL.

### Next Steps

- Learn how [Package References](/concepts/package-references) work
- Understand [Version Selectors](/concepts/version-selectors) for resolving packages
- See the full list of [Artifact Types](/concepts/artifact-types)
