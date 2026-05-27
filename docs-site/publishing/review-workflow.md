# Review Workflow

## Why

Before a package version reaches consumers, it needs human review. A maintainer might accidentally include a secret, write incorrect instructions, or miss an important skill. The review workflow ensures that at least one other person validates the content before it becomes immutable and published.

## How

### Review Lifecycle

```
draft → submit → automated checks → human approval → published
```

### Roles

| Role | Can | Cannot |
|---|---|---|
| **Maintainer** | Create drafts, upload artifacts, submit for review, publish | Approve their own submissions unless self-approval is enabled |
| **Reviewer** | Approve or request changes, add comments | Publish versions |

### Step 1: Submit for Review

After uploading all artifacts and confirming the manifest validates:

```bash
POST /api/v1/reviews/nwks/platform/agent-backend/versions/1.0.0/submit
Authorization: Bearer <token-with-review:write>
```

The version transitions from `draft` to `review`.

### Step 2: Automated Checks

Immediately after submission, the system runs:

**Blocking checks** (must pass before approval):

- Manifest schema validation
- Route/version consistency
- Artifact path validation
- Size limit enforcement
- Required metadata checks
- High-confidence secret scanning
- High-risk unsafe instruction scanning

**Warning checks** (advisory, don't block):

- Broken external links
- Additional advisory checks can be added later

### Step 3: Human Review

A reviewer opens `/reviews` in the web UI or calls the review queue API:

```bash
GET /api/v1/reviews
Authorization: Bearer <token-with-review:write>
```

The review queue includes submitted versions that are not published yet, including packages that do not appear on the public package list. A reviewer opens the submitted version and sees:

- Full manifest content
- Artifact file list with sizes and types
- Automated check results (pass/fail/warning)
- Markdown-rendered artifact previews
- Review history and comments

The reviewer can:

**Approve:**

```bash
POST /api/v1/reviews/{reviewId}/approve
{
  "comment": "Manifest looks correct, all artifacts present."
}
```

**Request changes:**

```bash
POST /api/v1/reviews/{reviewId}/request-changes
{
  "comment": "The AGENTS.md references a skill that doesn't exist in the package."
}
```

Requesting changes returns the version to `draft`. The maintainer must fix the issues and resubmit.

Maintainers can also explicitly reset an unpublished review version back to `draft` before reuploading content:

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/reset-draft
Authorization: Bearer <token-with-package:write>
```

This clears existing review state and approvals. It cannot be used on `published`, `deprecated`, or `yanked` versions.

### Step 4: Publish

After at least one approval, the maintainer publishes:

```bash
POST /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/publish
Authorization: Bearer <token-with-version:publish>
```

Publishing:

- Computes the package digest
- Locks all artifact blobs
- Updates the `@latest` alias pointer
- Writes an audit event

### Key Rules

| Rule | Description |
|---|---|
| **Self-approval blocked by default** | The same actor cannot approve their own submission unless `TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true` |
| **One approval required** | MVP requires at least one human approval before publish |
| **Content changes reset reviews** | Modifying a submitted version returns it to draft and invalidates approvals |
| **Reviewed content is locked** | Content changes require returning the version to draft and resubmitting |
| **Reviewer ≠ publisher** | The reviewer approves; the maintainer decides when to publish |

### Review Comments

MVP review comments are simple, version-level comments attached to review decisions:

```bash
POST /api/v1/reviews/{reviewId}/comments
{
  "body": "Consider adding a context-pack for the database schema."
}
```

Threaded comments and line-level artifact comments are deferred.

### Viewing Reviews

```bash
# List submitted versions awaiting review
GET /api/v1/reviews

# List all reviews for a version
GET /api/v1/reviews/nwks/platform/agent-backend/versions/1.0.0

# Get approval status
GET /api/v1/reviews/nwks/platform/agent-backend/versions/1.0.0/approval-status
```

### Local Browser Workflow

For single-user local development, enable self-approval before starting the server:

```bash
TROVE_REVIEWS_ALLOW_SELF_APPROVAL=true
```

Then push a package, open `/reviews`, approve it yourself, and publish it from the browser.

### Next Steps

- Learn about [Security Scanning](/publishing/security-scanning) that runs during review
- See the full [Upload & Publish Flow](/publishing/upload-publish-flow)
