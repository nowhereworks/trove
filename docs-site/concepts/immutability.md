# Immutability

## Why

Agents depend on deterministic content. If `nwks/platform/agent-backend@1.0.0` returns different bytes today than it did yesterday, builds break, behavior changes, and reproducibility is lost. Published versions in Trove are immutable — once published, the content cannot change.

## How

### What is Locked

When a version is published, the system locks:

- Artifact file content (blob bytes)
- Manifest content (`trove.yaml`)
- Version number
- Package digest
- Per-file digests and sizes

### Digest Computation

The package digest covers:

1. Canonical manifest JSON (sorted keys, stable formatting)
2. Sorted artifact paths
3. For each artifact: type, target path, file digest, and size

This means any change to the manifest or any artifact produces a different digest.

### Enforcement Layers

Immutability is enforced at two levels:

| Layer | Mechanism | Purpose |
|---|---|---|
| Application | API checks reject mutations | Friendly error messages with actionable codes |
| Database | PostgreSQL triggers and constraints | Safety against bypasses at the storage level |

### What Happens When You Need a Fix

You cannot edit a published version. Instead, publish a new version:

```
nwks/platform/agent-backend@1.0.0  ← published, immutable
nwks/platform/agent-backend@1.0.1  ← fix published as new version
nwks/platform/agent-backend@stable ← now points to 1.0.1
```

### Deprecating and Yanking

If a version has a problem:

- **Deprecate** — marks the version as discouraged but still resolvable. Use for versions that work but have a better replacement.
- **Yank** — hides the version from `@latest` and `@stable` resolution. The version remains fetchable by exact version or digest for authorized clients, so locked projects stay reproducible.

### Channel Updates

Channel aliases (`latest`, `stable`) are mutable pointers stored in a separate table. Publishing a new version updates the relevant channel pointers without touching the immutable version content.

### Example: Immutability in Action

```bash
# This works — creating a new version
POST /api/v1/packages/nwks/platform/agent-backend/versions
{ "version": "1.0.1" }

# This fails — trying to modify a published version
PUT /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/artifacts/AGENTS.md
# → 409 VERSION_IMMUTABLE: Published package versions are immutable.
```

### Next Steps

- See the full [Upload & Publish Flow](/publishing/upload-publish-flow)
- Learn how [Lockfiles](/cli/lockfiles) pin exact immutable versions
