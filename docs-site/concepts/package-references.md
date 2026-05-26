# Package References

## Why

Agents and CI systems need deterministic, unambiguous package identities. Shorthand references like `agent-backend` or `platform/agent-backend` are ambiguous — they depend on context that agents don't have. Trove requires full references everywhere so that manifests, lockfiles, API calls, and CLI commands are always deterministic.

## How

### Format

Every package reference uses this format:

```
{org}/{namespace}/{package}@{selector}
```

Examples:

```
nwks/platform/agent-backend@1.0.0
nwks/platform/agent-backend@latest
nwks/frontend/react-defaults@latest
nwks/ai/security-reviewer@sha256:abc123...
```

### Slug Rules

Organization, namespace, and package slugs follow the same pattern:

```
^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$
```

Rules:

- Lowercase letters, digits, and hyphens only
- Must start with a letter or digit
- Must end with a letter or digit
- Between 3 and 64 characters
- Slugs are compared case-sensitively (lowercase required)

### Display Names vs Slugs

Slugs are the immutable identity. Display names are human-readable labels that can change:

| Field | Example | Can Change? |
|---|---|---|
| `slug` | `agent-backend` | No — immutable after creation |
| `displayName` | `Backend Agent Defaults` | Yes — can be updated anytime |

### Full References Required

The following must always use the full `org/namespace/package` form:

- Package manifests (`trove.yaml`)
- Lockfiles (`.trove.lock.yaml`)
- CLI commands (`trove resolve`, `trove install`, etc.)
- API paths (`/api/v1/packages/{org}/{namespace}/{package}`)
- Raw URLs (`/raw/{org}/{namespace}/{package}/{path}@{selector}`)

Shorthand references are not supported in the MVP.

### Valid vs Invalid Examples

| Reference | Valid? | Reason |
|---|---|---|
| `nwks/platform/agent-backend@1.0.0` | Yes | Full reference with exact version |
| `nwks/platform/agent-backend@latest` | Yes | Full reference with named alias |
| `platform/agent-backend@1.0.0` | No | Missing org |
| `agent-backend@1.0.0` | No | Missing org and namespace |
| `nwks/Platform/agent-backend@1.0.0` | No | Namespace must be lowercase |
| `nwks/platform/Agent-Backend@1.0.0` | No | Package name must be lowercase |

### Next Steps

- Learn how [Version Selectors](/concepts/version-selectors) resolve to exact versions
- Understand [Lifecycle States](/concepts/lifecycle-states) for package versions
