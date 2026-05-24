# UI Requirements

## Stack

Default UI stack:

- React
- Vite
- TypeScript
- React Router
- TanStack Query
- Radix UI
- Tailwind CSS
- CodeMirror or Monaco for read-only file viewing and later editing

The UI must compile to static assets embedded in the Go binary.

Avoid a Next.js runtime unless the single-binary deployment goal changes.

## Brand Direction

`docs/trove_one_page_brand_kit_for_agents.md` is the source of truth for Trove visual design, logo direction, colors, typography, iconography, component styling, and product copy tone.

UI changes must follow the brand kit's secure developer-infrastructure direction: dark, precise, durable, practical, self-hosted, technical, and easy to scan.

## MVP Screens

- Login/logout.
- Organization and namespace list.
- Package search and browse.
- Adoption dashboard.
- Package detail.
- Version detail.
- Manifest viewer.
- Raw URL copy controls.
- Artifact file browser.
- Markdown renderer.
- Version diff viewer.
- Package/version creation.
- Draft artifact upload.
- Submit for review.
- Review and approval queue.
- Audit event view for package/version.

## Upload-Only MVP

MVP supports uploading/importing artifact content through the UI.

MVP does not include browser-based artifact editing. Browser editing can be added later once upload, validation, review, and publish flows are stable.

## Package Detail Requirements

Show:

- display name
- org, namespace, and package name
- description
- lifecycle
- visibility
- latest version
- stable version
- maintainers
- labels
- compatibility summary
- install snippets
- raw URL examples
- changelog
- versions list
- adoption counts by version when authorized

## Search Requirements

MVP search uses PostgreSQL-backed package search.

Default search shows published active packages only.

Required filters:

- organization
- namespace
- artifact type
- language label
- framework label
- tool compatibility
- visibility
- lifecycle

Do not search full artifact file contents in MVP.

## Adoption Dashboard Requirements

Show aggregate adoption counts by package and version.

Detailed project identity is visible only to authorized org/package users. Public package views must default to aggregate counts, not downstream project names or repository URLs.

## Version Detail Requirements

Show:

- exact version
- digest
- lifecycle state
- channel aliases pointing at the version
- publish timestamp
- publisher
- manifest
- artifact list
- per-file digest and size
- raw URL for each artifact
- review status
- audit trail

## Upload Flow

1. Maintainer creates a draft version.
2. Maintainer uploads package files or archive.
3. Server validates paths and extracts `trove.yaml`.
4. Server runs manifest validation.
5. UI shows validation errors inline.
6. Maintainer submits the draft for review.

## Review Flow

1. Reviewer opens submitted version.
2. UI shows manifest, artifact list, diffs, checks, and changelog.
3. Reviewer approves or requests changes with a simple general comment.
4. Maintainer publishes after required approvals pass.

MVP review comments are simple version-level comments attached to review decisions. Threaded comments and line-level artifact comments are deferred.

If a submitted version changes, the version returns to draft and existing approvals plus automated check results are invalidated.

Publishing requires one human approval by default. The maintainer controls final publish timing after approval gates pass.

## Accessibility And Responsiveness

- All MVP screens must be usable on desktop and mobile widths.
- Raw URL and install snippets need copy buttons with accessible labels.
- Search and artifact navigation must be keyboard usable.
