---
name: find-trove-skills
description: Helps users discover and install Trove-hosted agent skills when they ask for reusable agent capabilities.
---

# Find Trove Skills

Adapted from https://github.com/vercel-labs/skills/blob/main/skills/find-skills/SKILL.md

This skill helps you discover and install skills from Trove-hosted agent artifact registries.

## When to Use This Skill

Use this skill when the user:

- Asks "how do I do X" where X might be a common task with an existing skill
- Says "find a skill for X" or "is there a skill for X"
- Asks "can you do X" where X is a specialized capability
- Expresses interest in extending agent capabilities
- Wants to search for tools, templates, or workflows
- Mentions they wish they had help with a specific domain, such as design, testing, or deployment

## What is Trove Skills Search?

Trove skills search uses `trove skills find [query]` to discover packages with `skill` artifacts in your configured Trove registry. Skills are modular packages that extend agent capabilities with specialized knowledge, workflows, and tools.

Key commands:

- `trove skills find [query]` - Search for Trove-hosted skills by keyword
- `trove install <org/namespace/package@selector> --optional` - Install a skill package and include optional artifacts
- `trove check` - Check installed packages for updates
- `trove update` - Dry-run updates, or apply them with `--apply`

Browse skills in the Trove web UI at `/search?artifactType=skill`, open package pages at `/packages/{org}/{namespace}/{package}`, or call `/api/v1/search/packages?q=...&artifactType=skill` directly.

## How to Help Users Find Skills

### Step 1: Understand What They Need

When a user asks for help with something, identify:

1. The domain, such as React, testing, design, or deployment
2. The specific task, such as writing tests, creating animations, or reviewing PRs
3. Whether this is a common enough task that a reusable skill likely exists

### Step 2: Search for Skills

Run the find command:

```bash
trove skills find [query]
```

For example:

- User asks "how do I make my React app faster?" -> `trove skills find react performance`
- User asks "can you help me with PR reviews?" -> `trove skills find pr review`
- User asks "I need to create a changelog" -> `trove skills find changelog`

If no query is provided, `trove skills find` searches for `skill`.

### Step 3: Verify Quality Before Recommending

Do not recommend a skill based solely on search results. Always verify:

1. The package description matches the user's need
2. The source org and namespace are trusted for the user's environment
3. The package page and manifest show maintainers and a published stable version
4. The artifact paths and target paths are appropriate before installing optional artifacts

### Step 4: Present Options to the User

When you find relevant skills, present them to the user with:

1. The package reference and what it does
2. The exact install command
3. The package page path in the Trove web UI

Example response:

```text
I found a skill that might help: nwks/platform/react-best-practices@stable.
It provides React and Next.js performance optimization guidelines.

To install it:
trove install nwks/platform/react-best-practices@stable --optional

Learn more: /packages/nwks/platform/react-best-practices
```

### Step 5: Offer to Install

If the user wants to proceed, install the skill package:

```bash
trove install <org/namespace/package@selector> --optional
```

Use `--optional` because skill artifacts may be optional in mixed packages.

## Common Skill Categories

When searching, consider these common categories:

| Category | Example Queries |
|---|---|
| Web Development | react, nextjs, typescript, css, tailwind |
| Testing | testing, jest, playwright, e2e |
| DevOps | deploy, docker, kubernetes, ci-cd |
| Documentation | docs, readme, changelog, api-docs |
| Code Quality | review, lint, refactor, best-practices |
| Design | ui, ux, design-system, accessibility |
| Productivity | workflow, automation, git |

## Tips for Effective Searches

1. Use specific keywords: `react testing` is better than just `testing`
2. Try alternative terms: if `deploy` does not work, try `deployment` or `ci-cd`
3. Filter by type through the CLI or API so results include skill artifacts

## When No Skills Are Found

If no relevant skills exist:

1. Acknowledge that no existing Trove skill was found
2. Offer to help with the task directly using general capabilities
3. Suggest the user could package a reusable skill in Trove if this is recurring work

Example:

```text
I searched for Trove skills related to "xyz" but did not find any matches.
I can still help you with this task directly. Would you like me to proceed?
```
