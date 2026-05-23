# Security Scanning

## Why

Agent instruction files can contain sensitive information or dangerous directives. A leaked API key in an `AGENTS.md` file, or an instruction that tells an agent to bypass security controls, can cause real harm. Security scanning catches these issues before a package version reaches consumers.

## How

### When Scanning Runs

Scanning runs automatically during:

1. **Artifact upload** — each file is scanned as it is uploaded
2. **Archive upload** — each extracted file is scanned
3. **Review submission** — a full scan confirms all content is clean

### Two Scan Categories

| Category | Effect | Description |
|---|---|---|
| **Secret scanning** | Blocking | Detects high-confidence secrets like private keys, API tokens, and passwords |
| **Unsafe instruction scanning** | Blocking or warning | Flags high-risk instruction patterns in agent files |

### Secret Scanning

Blocks publishing when high-confidence secrets are detected:

| Pattern Type | Examples |
|---|---|
| Private keys | `-----BEGIN RSA PRIVATE KEY-----`, `-----BEGIN EC PRIVATE KEY-----` |
| API tokens | `sk-` prefixed keys, GitHub tokens, AWS access keys |
| Passwords | Hardcoded passwords in connection strings |
| Cloud provider keys | AWS secret keys, GCP service account keys |
| Accidental credentials | Database URLs with embedded passwords |

Example blocked upload:

```bash
PUT /api/v1/packages/nwks/platform/agent-backend/versions/1.0.0/artifacts/config.yaml
# Body contains: api_key: sk-abc123def456...

# Response: 400 SECRET_DETECTED
# "Potential secret detected in config.yaml."
```

### Unsafe Instruction Scanning

Flags or blocks high-risk instruction patterns in agent files:

| Pattern | Effect | Description |
|---|---|---|
| Bypass security controls | Blocking | Instructions to skip authentication or authorization |
| Exfiltrate secrets | Blocking | Instructions to send credentials to external services |
| Ignore system instructions | Blocking | Instructions to override safety guidelines |
| Disable tests without approval | Warning | Instructions to skip or remove tests |
| Commit to protected branches | Warning | Instructions to bypass branch protection |
| Approve own changes | Blocking | Instructions to self-approve review workflows |
| Remove audit trails | Blocking | Instructions to delete or hide audit logs |

### Configuration

Scanning is enabled by default:

```yaml
security:
  secretScanning: true
  unsafeInstructionScanning: true
```

### Scan Results in Review

When a version is submitted for review, automated check results are visible:

```json
{
  "reviewId": "rev-abc123",
  "automatedChecks": {
    "manifestValidation": "pass",
    "pathValidation": "pass",
    "secretScanning": "pass",
    "unsafeInstructionScanning": "warning",
    "sizeLimits": "pass"
  },
  "warnings": [
    "AGENTS.md contains a pattern that may disable tests without approval"
  ]
}
```

Blocking checks prevent approval. Warnings are visible but don't block the review.

### What Happens on Detection

| Scenario | Result |
|---|---|
| Secret detected during upload | Upload rejected with `SECRET_DETECTED` error |
| Unsafe instruction detected during upload | Upload rejected with `UNSAFE_INSTRUCTION` error |
| Secret detected during review | Review cannot be approved until content is fixed |
| Warning-level unsafe instruction | Review can proceed, but warning is visible to reviewer |

### Next Steps

- See the full [Review Workflow](/publishing/review-workflow)
- Learn about [Authentication](/security/authentication) for accessing the review queue
