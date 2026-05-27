# What is Trove?

::: warning Pre-v1.0 Software
Trove is pre-v1.0 software. It is highly unstable and subject to heavy changes, including additions, removals, and breaking changes to functionality.

The API contract is not guaranteed at this stage. Use at your own risk.
:::

Trove is a self-hosted registry for the files that tell agents, CLIs, and developer tools how to work inside your organization. Instead of copying `AGENTS.md` files, reusable skills, prompts, commands, review rubrics, and workflow instructions across repositories, teams publish them as versioned packages that can be searched, reviewed, installed, locked, and updated.

The problem Trove solves is drift. Without a registry, every repo slowly grows its own copy of the same operating knowledge: some files are outdated, some are unreviewed, some are impossible to trace, and no one can tell which projects depend on which instructions. Trove gives platform and security teams one controlled place to distribute blessed agent artifacts while still giving developers and agents a simple way to resolve, fetch, and pin the exact version they use.

In practical terms, Trove works like a lightweight internal package registry for agent-facing engineering context. Packages live under `org/namespace/package`, published versions are immutable, access is governed by visibility and RBAC, and the CLI/API can install artifacts into projects, check for updates, and report adoption. The result is repeatable agent behavior, auditable publishing, and less hand-maintained instruction sprawl across repositories.

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
| **Discovery** | Search packages by name, description, artifact type, and more |
| **Agent APIs** | Resolve selectors to exact versions, fetch raw artifacts, download archives |
| **CLI** | Install, check for updates, and manage lockfiles from the command line |
| **Governance** | RBAC, review workflows, approval gates, security scanning, and audit trails |
| **Adoption Tracking** | See which projects use your packages through CLI-reported install data |

### Who Uses Trove

- **Platform engineers** publish and maintain blessed artifact packages
- **Application developers** discover and adopt packages for their projects
- **Agents and CLIs** resolve versions, fetch raw files, and check for updates
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

The binary serves JSON APIs, raw artifact endpoints, an embedded React SPA, and health endpoints. All metadata and artifact blobs are stored in PostgreSQL.

### Next Steps

- Learn how [Package References](/concepts/package-references) work
- Understand [Version Selectors](/concepts/version-selectors) for resolving packages
- See the full list of [Artifact Types](/concepts/artifact-types)
