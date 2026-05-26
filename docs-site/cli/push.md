# trove push

## Why

Use `trove push` to run the complete `AGENTS.md` publishing workflow: create or reuse a draft, upload `trove.yaml`, upload `AGENTS.md`, publish, or submit for review when approval is required.

If the target org exists, the server can auto-create a missing namespace and package during draft creation. Operators control this with server-side `TROVE_CREATE_NAMESPACE_ON_PUSH` and `TROVE_CREATE_PACKAGE_ON_PUSH`; both default to enabled.

## How

### Publish The Default Patch Version

```bash
export TROVE_TOKEN=trove_token_...
trove push
```

When publishing succeeds:

```text
Published nwks/platform/agent-defaults@1.0.0
Install with:
  trove install nwks/platform/agent-defaults@latest
```

When review approval is required, the default mode submits for review instead of failing:

```text
Uploaded nwks/platform/agent-defaults@1.0.0
Submitted for review:
  https://trove.company.com/reviews
```

### Choose A Version

```bash
trove push --patch
trove push --minor
trove push --major
trove push --version 1.2.3
```

### Control Publishing Mode

```bash
trove push --draft
trove push --submit-only
trove push --publish
```

`--publish` fails if review approval blocks publication. Default mode catches only the structured `APPROVAL_REQUIRED` error and submits for review.

### Overwrite An Unpublished Version

```bash
trove push --force --version 1.0.0
```

`--force` resets an existing `review` version back to `draft`, clears prior review state, and reuploads `trove.yaml` and `AGENTS.md`. Existing `draft` versions are already mutable and are reused by default.

`--force` never overwrites `published`, `deprecated`, or `yanked` versions. Use a new SemVer version for immutable versions.

### JSON Output

```bash
trove push --json
```

When review is required, JSON output reports the submitted review state:

```json
{
  "package": "nwks/platform/agent-defaults",
  "version": "1.0.0",
  "lifecycle": "review",
  "reviewUrl": "https://trove.company.com/reviews?package=nwks%2Fplatform%2Fagent-defaults&version=1.0.0",
  "installCommand": "trove install nwks/platform/agent-defaults@latest"
}
```

## Reference

| Flag | Required | Description |
|---|---|---|
| `--patch` | No | Select next patch version; default |
| `--minor` | No | Increment minor and reset patch |
| `--major` | No | Increment major and reset minor and patch |
| `--version <x.y.z>` | No | Use an explicit strict SemVer version |
| `--remote <name>` | No | Push to a configured remote |
| `--visibility private\|internal\|public` | No | Override visibility for this push |
| `--draft` | No | Upload draft only |
| `--submit-only` | No | Upload and submit for review |
| `--publish` | No | Require publish to succeed |
| `--force` | No | Reset an unpublished review version to draft before upload |
| `--json` | No | Emit machine-readable output |

`trove push` never rewrites `AGENTS.md`. It may rewrite generated fields in `trove.yaml` so the uploaded manifest matches the selected remote and version.

`trove push` does not create orgs. Create the org first or configure `TROVE_ORG` on the server to bootstrap it at startup.

## Next

- [review workflow](/publishing/review-workflow) explains approval behavior
- [install](/cli/install) consumes the published package
