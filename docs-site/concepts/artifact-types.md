# Artifact Types

## Why

Trove manages many kinds of agent-consumable files. Categorizing them by type enables filtering in search and install-time selection. Instead of treating every file as a generic blob, Trove recognizes 10 first-class artifact types.

## How

### Artifact Types Reference

| Type | Example Path | Purpose |
|---|---|---|
| `agent-instructions` | `AGENTS.md` | Repo-level agent behavior and engineering rules |
| `skill` | `skills/api-review/SKILL.md` | Reusable focused capability for agents |
| `command` | `commands/review.md` | Slash-command-style workflow definition |
| `subagent` | `agents/security-reviewer.md` | Specialized agent role or persona |
| `context-pack` | `context/kubernetes.md` | Domain knowledge bundle |
| `template` | `templates/backend-api/**` | Project or artifact scaffolding |
| `rubric` | `rubrics/pr-review.md` | Review criteria and evaluation rubrics |
| `policy` | `policies/model-routing.yaml` | Tool, model, and runtime rules |
| `prompt` | `prompts/incident-analysis.md` | Reusable prompt patterns |
| `adapter` | `adapters/opencode.md` | Mapping for a specific agent harness |

### Declaring Artifact Types

Artifact types are declared in `Trovefile`:

```yaml
spec:
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
    - path: skills/api-review/SKILL.md
      type: skill
      required: false
      targetPath: .opencode/skills/api-review/SKILL.md
    - path: commands/review.md
      type: command
      required: false
      targetPath: .opencode/commands/review.md
```

### Required vs Optional

- `required: true` — installed by default with `trove install`
- `required: false` — must be explicitly selected during install

### Type Validation

Publishing fails when:

- An artifact uses an unknown type
- A required artifact file is missing from the uploaded package
- Artifact paths are duplicated within a version

### Searching by Type

Search supports filtering by artifact type:

```
GET /api/v1/search/packages?q=agent&artifactType=skill
```

This returns packages that declare at least one artifact of the specified type.

### Example: A Full Package

```
nwks/platform/agent-backend@1.0.0
├── AGENTS.md                          (agent-instructions, required)
├── skills/backend-api/SKILL.md        (skill, optional)
├── commands/lint.md                   (command, optional)
├── agents/security-reviewer.md        (subagent, optional)
├── context/backend-patterns.md        (context-pack, optional)
└── policies/go-routing.yaml           (policy, optional)
```

### Next Steps

- Learn how to declare artifacts in [Manifests](/publishing/manifests)
- See how to install specific artifacts with [trove install](/cli/install)
