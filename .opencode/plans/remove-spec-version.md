# Remove spec.version from trove.yaml

## Summary
Remove `spec.version` from the manifest format. The database is the authoritative source of truth for version. Version is provided separately during publish via API request body and URL path.

## Changes

### 1. `internal/manifest/manifest.go`
- Remove `Version string` field from `Spec` struct (line 35)
- Remove `requireSemver(&problems, "spec.version", m.Spec.Version)` call (line 130)
- Remove version mismatch check block (lines 143-145)
- Remove `Version string` from `ValidateOptions` struct (line 74)

### 2. `internal/manifest/manifest_test.go`
- Remove `"spec.version"` from required fields list in `TestValidateReportsRequiredFields` (line 31)
- Remove `"spec.version"` from route mismatch fields in `TestValidateReportsRouteMismatches` (line 45)
- Remove `m.Spec.Version = "1.0"` in `TestValidateRejectsInvalidSlugsAndSemver` (line 55)
- Remove `"spec.version"` from problem fields list (line 60)
- Remove `version: 1.0.0` from `validManifestYAML` fixture (line 150)

### 3. `internal/cli/project.go`
- Remove `Version: "1.0.0"` from `generatedAgentsManifest` Spec literal (line 396)
- Remove `m.Spec.Version = version` from `applyGeneratedManifestFields` (line 174)

### 4. `internal/packages/postgres_store.go`
- Remove `Version: version.Version` from `ValidateOptions` in `UploadArtifact` (line 148)
- Remove `Version: version.Version` from `ValidateOptions` in `PublishVersion` (line 229)

### 5. `internal/cli/push_test.go`
- Remove `m.Spec.Version != "1.0.0"` assertion, keep only lifecycle check (lines 39-41)
- Remove `"version: 1.0.0"` / `"version: 1.0.1"` from `wantBodyContains` in mock upload handlers (lines 215, 217)

### 6. `internal/cli/agents_md_test.go`
- Remove `m.Spec.Version = strings.TrimPrefix(version, "v")` from `agentsManifest()` (line 245)

### 7. `internal/api/router_test.go`
- Remove `body.Spec.Version != "1.0.0"` from `TestManifestEndpointReturnsManifestJSON` assertion (line 592)
- Remove `version: 1.0.1` from `sliceTwoManifestYAML` fixture (line 824)

### 8. `docs-site/publishing/manifests.md`
- Remove `version: 1.0.0` from full example YAML (line 27)
- Remove `spec.version` row from Required Fields table (line 69)
- Remove version mismatch validation rule row (line 105)

### 9. `docs-site/cli/push.md`
- Update sentence about rewriting trove.yaml to remove "version" mention (line 97)

## Verification
- `go test ./...` must pass
- `go build ./...` must succeed
