# Upload & Publish Flow

## Why

Publishing a package isn't a single action — it's a workflow. You create a draft, upload artifacts, validate the manifest, submit for review, get approval, and then publish. Each step has guards to ensure quality and prevent mistakes.

## How

### AGENTS.md CLI Flow

For the standard repo-root `AGENTS.md` package, use the CLI. It hides draft creation, manifest upload, review submission, and publishing behind a Git-like flow:

```bash
trove init agents-md \
  --remote https://trove.company.com/nwks/platform/agent-defaults
trove status
trove push
```

`trove push` uploads `Trovefile` first, uploads `AGENTS.md` second, then publishes or submits for review if approval is required.

If approval is required, open `/reviews` in the web UI to approve and publish the submitted version. For single-user local development, start the server with `TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true` to approve your own submissions in the browser.

### Browser Flow

The web UI can run the same workflow without leaving the browser:

1. Open `/upload`
2. Create the package version draft
3. Upload an archive or paste the `Trovefile`
4. Click publish
5. If approval is required, follow the link to `/reviews`
6. Approve, then publish the version from the review queue

### Raw API Flow

Use the raw API when building automation that cannot shell out to the CLI.

### Step-by-Step Flow

```
1. Create draft version
2. Upload artifacts (.tar.gz, .zip, or individual files)
3. Server validates manifest and artifacts
4. Fix any validation errors
5. Submit for review
6. Reviewer approves (or requests changes)
7. Maintainer publishes
```

### Step 1: Create a Draft

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions
Content-Type: application/json
Authorization: Bearer <token>

{
  "version": "1.0.0",
  "visibility": "private"
}
```

Response:

```json
{
  "id": "01901234-5678-9abc-def0-123456789abc",
  "org": "nwks",
  "namespace": "platform",
  "package": "agent-backend",
  "version": "1.0.0",
  "lifecycle": "draft",
  "visibility": "private",
  "createdAt": "2026-05-23T00:00:00Z"
}
```

### Step 2: Upload Artifacts

You can upload in two ways:

**Option A: Upload an archive**

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/archive
Content-Type: application/gzip
Authorization: Bearer <token>

<binary: package.tar.gz>
```

The server extracts the archive, validates paths, and stores individual artifact files.

**Option B: Upload individual files**

```bash
PUT /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/artifacts/Trovefile
Content-Type: application/yaml
Authorization: Bearer <token>

apiVersion: trove.io/v1
kind: TrovePackage
# ... manifest content ...
```

```bash
PUT /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/artifacts/AGENTS.md
Content-Type: text/markdown
Authorization: Bearer <token>

# Backend Agent Instructions
...
```

### Step 3: Server Validation

After upload, the server automatically validates:

- Manifest schema and required fields
- Route consistency (manifest org/namespace/name must match the URL)
- Artifact path rules (no `..`, no duplicates, no escaping)
- Size limits (per-file and total package)
- Secret scanning patterns
- Unsafe instruction patterns

Validation errors are returned inline in the UI or as API errors.

### Step 4: Fix Errors

If validation fails, fix the manifest or artifacts and re-upload. Draft versions are mutable — you can replace any artifact before submission.

### Step 5: Submit for Review

```bash
POST /api/v1/reviews/nwks/platform/agent-backend/versions/1.0.0/submit
Authorization: Bearer <token>
```

The version moves from `draft` to `review`. Automated checks run, and the version appears in the review queue.

### Step 6: Review

A reviewer opens `/reviews` in the UI or lists the review queue by API:

```bash
GET /api/v1/reviews
Authorization: Bearer <token-with-review:write>
```

The reviewer opens the version and:

- Reviews the manifest
- Checks artifact content
- Sees automated scan results
- Approves or requests changes

Self-approval is blocked by default. Enable `TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true` for local single-user review workflows.

```bash
# Approve
POST /api/v1/reviews/{reviewId}/approve
{ "comment": "Looks good, follows our patterns." }

# Request changes
POST /api/v1/reviews/{reviewId}/request-changes
{ "comment": "Missing required skill for database migrations." }
```

If changes are requested, the version returns to `draft` and existing approvals are invalidated.

### Step 7: Publish

After approval, the maintainer publishes:

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/publish
Authorization: Bearer <token>
```

The version becomes immutable and an audit event is written.

### Size Limits

| Limit | Default | Config Key |
|---|---|---|
| Max artifact file size | 10 MiB | `storage.limits.maxArtifactFileBytes` |
| Max unpacked package size | 100 MiB | `storage.limits.maxUnpackedPackageBytes` |
| Max artifacts per version | 1,000 | `storage.limits.maxArtifactsPerVersion` |

### Next Steps

- Understand the [Review Workflow](/publishing/review-workflow) in detail
- Learn about [Security Scanning](/publishing/security-scanning) that runs during upload
