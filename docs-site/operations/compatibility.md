# Compatibility

## Why

Not every package works with every tool, model, or runtime. A skill designed for OpenCode 1.0 might not work with OpenCode 0.6. A prompt that requires 128K context won't work with a 4K model. Compatibility metadata lets Trove warn users before they install something that won't work.

## How

### Compatibility Schema

Packages declare compatibility in their manifest:

```yaml
spec:
  compatibility:
    tools:
      - name: opencode
        version: ">=0.6.0 <2.0.0"
    models:
      - family: gpt
        minContextWindow: 128000
    runtimes:
      - linux
      - darwin
```

### Tool Compatibility

| Field | Type | Description |
|---|---|---|
| `name` | string | Lowercase tool slug (e.g., `opencode`, `cursor`) |
| `version` | string | SemVer range (e.g., `>=0.6.0 <2.0.0`) |

Tool versions use SemVer range matching:

| Range | Matches |
|---|---|
| `>=0.6.0 <2.0.0` | 0.6.0, 0.7.1, 1.0.0, 1.9.9 |
| `^1.0.0` | 1.0.0, 1.1.0, 1.9.9 (not 2.0.0) |
| `~1.2.0` | 1.2.0, 1.2.1, 1.2.9 (not 1.3.0) |
| `>=1.0.0` | 1.0.0 and above |

### Model Compatibility

| Field | Type | Description |
|---|---|---|
| `family` | string | Lowercase model family (e.g., `gpt`, `claude`, `gemini`) |
| `minContextWindow` | integer | Minimum context window in tokens (optional) |

Model family matching is exact:

| Declared | Compatible With |
|---|---|
| `family: gpt` | `gpt-4`, `gpt-4o`, `gpt-3.5-turbo` |
| `family: claude` | `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku` |
| `family: gemini` | `gemini-1.5-pro`, `gemini-1.5-flash` |

Context window matching is a minimum threshold:

| Declared | Target Model | Compatible? |
|---|---|---|
| `minContextWindow: 128000` | 128K model | Yes |
| `minContextWindow: 128000` | 32K model | No |
| `minContextWindow: 128000` | 1M model | Yes |

### Runtime Compatibility

Runtimes are a list of named lowercase strings:

| Runtime | Description |
|---|---|
| `linux` | Linux operating system |
| `darwin` | macOS operating system |
| `windows` | Windows operating system |
| `kubernetes` | Kubernetes cluster environment |
| `docker` | Docker container environment |

Runtime matching is exact name matching. A package declaring `linux` is compatible with Linux targets only.

### Compatibility Check

The compatibility check API compares a package version against a target:

```bash
POST /api/v1/compatibility/check
Content-Type: application/json

{
  "package": "nwks/platform/agent-backend",
  "version": "1.0.0",
  "target": {
    "tool": "opencode",
    "toolVersion": "0.6.0",
    "runtime": "linux",
    "modelFamily": "gpt",
    "contextWindow": 128000
  }
}
```

Response:

```json
{
  "package": "nwks/platform/agent-backend",
  "version": "1.0.0",
  "compatibility": "compatible",
  "details": {
    "tool": "compatible",
    "runtime": "compatible",
    "model": "compatible"
  }
}
```

### Compatibility Values

| Value | Meaning |
|---|---|
| `compatible` | All constraints match the target |
| `incompatible` | At least one constraint doesn't match |
| `unknown` | No compatibility metadata declared |

### Strict vs Advisory Mode

| Mode | Behavior |
|---|---|
| **Advisory (default)** | `unknown` compatibility is informational; doesn't block installs or checks |
| **Strict** | `unknown` compatibility is treated as a failure; blocks installs and checks |

CLI flags:

```bash
# Advisory (default)
trove check --lock .trove.lock.yaml
# → "compatibility: unknown" — exits 0

# Strict
trove check --lock .trove.lock.yaml --strict-compatibility
# → "compatibility: unknown" — exits 1
```

### Unknown Compatibility

When a package doesn't declare compatibility metadata:

- The compatibility check returns `unknown`
- `trove check` reports it as advisory (default)
- `trove update` skips the update if `--strict-compatibility` is set
- The package is still installable

### Example: Incompatible Update

```bash
trove update --lock .trove.lock.yaml

nwks/ai/security-reviewer
  Current: 0.3.0
  Latest:  1.0.0
  Compatibility: incompatible (requires opencode >=1.0.0, you have 0.6.0)
  Skipping incompatible update.
```

### Next Steps

- See [Configuration](/operations/configuration) for server setup
- Learn how [trove check](/cli/check) reports compatibility
