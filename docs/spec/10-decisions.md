# Decisions

## Accepted Decisions

| Decision | Status | Rationale |
|---|---|---|
| Use internal name AgentHub | Accepted | Short, clear project name |
| Build a Go single binary | Accepted | Keeps deployment simple and self-hostable |
| Embed React/Vite static UI | Accepted | Fits single-binary deployment without Next.js runtime complexity |
| Use PostgreSQL for MVP metadata and blobs | Accepted | Queryable source of truth with simpler permissions than Git-backed storage |
| Store MVP artifact bytes in PostgreSQL `bytea` | Accepted | Simplest durable blob path for the first implementation slice |
| Target RustFS/S3-compatible storage soon after MVP | Accepted | Keeps self-hosting viable while providing a path for larger artifacts |
| Use `golang-migrate/migrate` for migrations | Accepted | SQL-first migrations fit the PostgreSQL MVP and Go binary deployment |
| Use `sqlc` with `pgx` for DB access | Accepted | SQL-first type-safe queries align with migration-owned PostgreSQL schema |
| Run startup migrations only when explicitly enabled | Accepted | Keeps local development easy while avoiding production startup migration risk |
| Enforce immutability with DB triggers plus app checks | Accepted | Friendly API errors plus database-level safety against bypasses |
| Use UUID v7 primary keys | Accepted | Globally unique and time-sortable IDs with better locality than random UUIDs |
| Use `timestamptz` UTC timestamps | Accepted | PostgreSQL-native timestamps with unambiguous API serialization |
| Test DB behavior against real PostgreSQL | Accepted | PostgreSQL-specific triggers, constraints, JSONB, arrays, and locking matter |
| Use `chi` for HTTP routing | Accepted | Small idiomatic router with path params and middleware without a large framework |
| Use one namespace level | Accepted | MVP path shape is `org/namespace/package`; nested namespaces are deferred |
| Make package names unique within namespace | Accepted | The full `org/namespace/package` tuple is the canonical package identity |
| Use lowercase DNS-like slugs | Accepted | Avoids URL, routing, and case-normalization ambiguity |
| Defer slug renames | Accepted | Prevents broken pinned references and redirect complexity in MVP |
| Require full package references | Accepted | Manifests, lockfiles, APIs, and CLI commands remain deterministic for agents |
| Require raw auth by default | Accepted | Enterprise-safe default |
| Support `private`, `internal`, and `public` visibility | Accepted | Covers private packages, installation-wide packages, and public collections |
| Allow public namespaces/packages | Accepted | Needed for GitHub-like public collections |
| Use most-restrictive visibility inheritance | Accepted | Prevents accidentally public packages under private parents |
| Allow anonymous reads for public metadata and artifacts | Accepted | Public packages need agent-friendly anonymous resolve, manifest, archive, and raw fetches |
| Use scoped opaque bearer tokens | Accepted | Allows revocation, auditing, and least-privilege CI/agent access |
| Use OIDC plus gated dev auth mode | Accepted | Keeps production enterprise-ready without blocking local development |
| Include MVP CLI prototype | Accepted | Agent and CI consumption should be practical early |
| Make `agenthub check` advisory for ordinary updates by default | Accepted | Avoids noisy CI failures while allowing strict policy flags |
| Use text CLI output with `--json` support | Accepted | Serves humans by default and agents/CI when structured output is needed |
| Install required artifacts by default | Accepted | Avoids surprising optional skill/command installs |
| Make updates dry-run first | Accepted | Shows diffs before changing repo instruction files |
| Treat unknown compatibility as advisory | Accepted | Early packages remain usable while still surfacing incomplete metadata |
| Canonically rewrite lockfiles | Accepted | Keeps generated lockfiles deterministic and implementation simple |
| Use upload-only UI for MVP content changes | Accepted | Avoids browser editor complexity before validation/review/publish flows stabilize |
| Use React Router and Radix UI | Accepted | Common React/Vite choices with flexible routing and accessible primitives |
| Require one approval before publish | Accepted | Provides governance without excessive MVP friction |
| Let maintainers publish after approval | Accepted | Maintainers own release timing after review gates pass |
| Reset review on submitted-content changes | Accepted | Prevents approved content from changing before publish |
| Block on security and validity checks | Accepted | Stops unsafe or malformed packages while keeping quality warnings non-blocking |
| Use simple review comments | Accepted | Supports requested changes without threaded/line-level review complexity |
| Use PostgreSQL full-text search for MVP | Accepted | Good enough for metadata discovery without OpenSearch infrastructure |
| Search published active packages by default | Accepted | Keeps normal discovery focused on usable packages |
| Index metadata and manifest fields for search | Accepted | Enables artifact/tool discovery without indexing artifact contents |
| Report adoption through configured CLI calls | Accepted | Reuses install/check/update flows without requiring a separate agent |
| Show aggregate adoption by default | Accepted | Gives maintainers signal while protecting downstream project privacy |
| Require core search filters | Accepted | Covers org, namespace, artifact type, labels, tool compatibility, visibility, and lifecycle |
| Use generic MVP compatibility schema | Accepted | Covers tools, models, and runtimes without per-harness schema complexity |
| Use SemVer ranges for tool compatibility | Accepted | Allows minimum and upper-bound tool compatibility checks |
| Use named runtime strings | Accepted | Keeps runtime matching simple for MVP |
| Use model family plus context window | Accepted | Captures important model constraints without brittle exact model IDs |
| Keep OIDC standards-based | Accepted | Avoids baking provider-specific assumptions into MVP auth |
| Make strict compatibility fail unknown | Accepted | Default remains advisory while strict CI/policy can block incomplete metadata |
| Require strict SemVer for published versions | Accepted | Enables deterministic major/minor selectors and compatibility-aware updates |
| Limit MVP aliases to `latest` and `stable` | Accepted | Covers common install/update flows without arbitrary channel governance |
| Published versions are immutable | Accepted | Required for deterministic agent behavior |
| Aliases resolve to exact versions | Accepted | Makes cache behavior and debugging cleaner |
| Redirect alias raw URLs to exact URLs | Accepted | Keeps caching semantics explicit and makes resolved versions visible |
| Keep yanked exact versions fetchable | Accepted | Preserves reproducibility for locked projects while hiding yanked versions from aliases |
| Use package-version digest selectors only | Accepted | `@sha256:<digest>` resolves package versions; file digests stay metadata |
| Generate archives on demand | Accepted | Avoids duplicate archive storage and keeps package identity normalized around files |
| Digest canonical manifest plus files | Accepted | Package digests change when manifest or artifact metadata/content changes |
| Support `.tar.gz` and `.zip` uploads | Accepted | Covers common CLI and GitHub-style package workflows |
| Use configurable size limits | Accepted | Protects PostgreSQL-backed blob storage while allowing operator tuning |
| Require manifest descriptions | Accepted | Improves discovery, review quality, and package pages |
| Require manifest maintainers | Accepted | Every published package needs explicit ownership |
| Default missing artifact `targetPath` to `path` | Accepted | Keeps simple packages concise while allowing install remapping |
| Treat manifest dependencies as declare-only in MVP | Accepted | Captures metadata without expanding install/publish scope |
| Include lockfile generator metadata | Accepted | Records tool name, version, and generation timestamp for debugging |
| Do not overwrite install targets by default | Accepted | Protects existing repo files and instructions from accidental replacement |
| Use cursor pagination for list APIs | Accepted | More stable than offsets for changing package/search lists |
| Use camelCase JSON fields | Accepted | Matches existing examples and TypeScript/browser clients |
| Return resource JSON from write endpoints | Accepted | Avoids extra client GETs after create or state transitions |
| Include `X-Request-Id` on every API response | Accepted | Improves tracing for success and error cases |
| Keep `package@selector` route segments | Accepted | Preserves ergonomic registry-like references |
| Registry does not directly edit downstream repos by default | Accepted | Repo changes should happen through CLI, CI, Backstage, or agent proposal workflows |

## Deferred Decisions

| Question | Default For Now |
|---|---|
| Git-backed import/export | Defer until after PostgreSQL-backed MVP works |
| S3/RustFS blob storage | Defer from Slice 1; design schema/config so it can be added soon |
| Native GitHub/GitLab PR integration | Defer; document CLI/CI flows first |
| Full artifact signing/provenance | Defer; keep digest model now |
| OpenSearch | Defer; start with PostgreSQL full-text search |
| Full artifact content search | Defer; MVP indexes metadata and manifest fields only |
| Tool-specific compatibility schemas | Defer; MVP uses generic tools/models/runtimes schema |
| OCI compatibility | Defer |
| Advanced policy language | Defer; start with explicit checks and RBAC |
| Tenant/customer overlays | Defer |
| Threaded or line-level review comments | Defer; MVP has simple version-level comments |
| Slug rename redirects | Defer; slugs are immutable in MVP |
| Prerelease or non-SemVer versions | Defer; MVP published versions are strict SemVer |
| Arbitrary user-managed channels | Defer; MVP supports `latest` and `stable` only |

## Remaining Open Questions

- None currently blocking Slice 1.
