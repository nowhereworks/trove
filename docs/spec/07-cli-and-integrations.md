# CLI And Integrations

## MVP CLI Prototype

The MVP includes a small CLI prototype for agent and CI flows. Trove-wide command naming, selector defaults, raw URL behavior, and worktree semantics are defined in [`12-command-semantics.md`](12-command-semantics.md).

Initial commands:

```bash
trove resolve companyx/platform/agent-backend@stable
trove download companyx/platform/agent-backend@stable AGENTS.md
trove install companyx/platform/agent-backend@stable --target .
trove check --lock .trove.lock.yaml
trove update --lock .trove.lock.yaml
trove update --lock .trove.lock.yaml --apply
```

The CLI can be implemented inside the Go binary as subcommands or as a separate `cmd/trove` entrypoint that shares packages with the server.

## CLI Behavior

- `resolve` calls the resolve API and prints the exact version and digest.
- `download` downloads one artifact to stdout or a target path.
- `fetch` may remain as an MVP compatibility alias for `download`.
- `install` downloads required artifacts by default and writes `.trove.lock.yaml`.
- `check` reads `.trove.lock.yaml` and calls the update check API.
- `update` shows available updates and diffs without writing files.
- `update --apply` writes changed artifacts and rewrites `.trove.lock.yaml`.
- `install`, `check`, and `update --apply` may report lockfile/adoption summaries when configured with an API token.

CLI commands default to human-readable text output. Commands that return structured data must support `--json` for agents and CI.

The registry should not directly edit downstream repositories by default. Changes happen through CLI, CI, Backstage, or agent proposal workflows.

`trove install` must not overwrite existing files that differ from the fetched artifact unless the caller passes an explicit overwrite option. Non-interactive callers should fail with a clear conflict error instead of prompting.

Manifest dependencies are declare-only in the MVP. The CLI may display dependency metadata, but it must not auto-install dependencies until dependency resolution is explicitly designed.

`trove install` installs only manifest artifacts marked `required: true` by default. Optional artifacts require explicit selection flags.

When `trove install` or `trove update --apply` changes the lockfile, it rewrites `.trove.lock.yaml` as canonical YAML with stable ordering. Comments and hand formatting are not preserved.

## OpenCode Init

Target UX:

```bash
opencode /init https://trove.company.com/companyx/platform/agent-backend@stable
```

The URL should resolve to package metadata, manifest, raw artifact URLs, and archive download links.

## CI Integration

Example CI flow:

```bash
trove check --lock .trove.lock.yaml
```

MVP behavior is advisory by default:

- report newer compatible versions
- report yanked installed versions
- report incompatible installed versions
- exit zero when only ordinary updates are available
- exit non-zero for yanked or incompatible installed versions
- exit non-zero for ordinary updates only when configured by flags such as `--fail-on-update` or CI policy

Missing or unsupported compatibility metadata is reported as `unknown`. Unknown compatibility is advisory by default and does not block installs or checks unless a policy or CLI flag requires strict compatibility.

Adoption reporting is opt-in by configuration/token. Reports send package references, versions, digests, artifact targets, project metadata, and timestamps; they do not send artifact file contents.

## Backstage Integration

Backstage can use Trove during scaffolding:

- resolve a default package by project type
- download a package archive
- copy target artifacts into the new repository
- write `.trove.lock.yaml`

## GitHub/GitLab Integration

MVP can document CI examples but does not need native GitHub/GitLab apps.

Later integrations can open pull requests with CLI-generated updates.
