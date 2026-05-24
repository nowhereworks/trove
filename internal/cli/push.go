package cli

import (
	"errors"
	"fmt"
	"os"
)

func RunPush(args []string, jsonOutput bool) error {
	remoteNameFlag := flagValue(args, "--remote")
	visibilityOverride := flagValue(args, "--visibility")
	channelOverride := flagValue(args, "--channel")
	explicitVersion := flagValue(args, "--version")
	bump := "patch"
	selected := 0
	for _, pair := range []struct{ flag, bump string }{{"--patch", "patch"}, {"--minor", "minor"}, {"--major", "major"}} {
		if hasFlag(args, pair.flag) {
			bump = pair.bump
			selected++
		}
	}
	if explicitVersion != "" {
		selected++
	}
	if selected > 1 {
		return fmt.Errorf("--patch, --minor, --major, and --version are mutually exclusive")
	}

	mode := "default"
	modeCount := 0
	for _, pair := range []struct{ flag, mode string }{{"--draft", "draft"}, {"--submit-only", "submit"}, {"--publish", "publish"}} {
		if hasFlag(args, pair.flag) {
			mode = pair.mode
			modeCount++
		}
	}
	if modeCount > 1 {
		return fmt.Errorf("--draft, --submit-only, and --publish are mutually exclusive")
	}

	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", projectConfigPath, err)
	}
	if problems := validateProjectConfig(cfg); len(problems) > 0 {
		return fmt.Errorf("invalid %s: %s", projectConfigPath, problems[0])
	}
	remoteName, remote, err := remoteForConfig(cfg, remoteNameFlag)
	if err != nil {
		return err
	}
	ref, err := ParsePackageRefNoSelector(remote.Package)
	if err != nil {
		return err
	}
	m, err := loadManifestYAML(manifestPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", manifestPath, err)
	}
	if _, err := os.Stat(agentsMDPath); err != nil {
		return fmt.Errorf("AGENTS.md is required: %w", err)
	}
	if os.Getenv("TROVE_TOKEN") == "" {
		return fmt.Errorf("TROVE_TOKEN is required to push")
	}

	client := NewClientForServer(remote.ServerURL)
	versions := []PackageVersion{}
	if detail, err := client.GetPackage(ref.Org, ref.Namespace, ref.Name); err == nil {
		versions = detail.Versions
	} else {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 404 {
			return fmt.Errorf("lookup package %s: %w", ref.PackagePath(), err)
		}
		_, err = client.CreatePackage(CreatePackageRequest{Org: ref.Org, Namespace: ref.Namespace, Name: ref.Name, DisplayName: firstNonEmpty(m.Metadata.DisplayName, titleFromSlug(ref.Name)), Description: firstNonEmpty(m.Metadata.Description, "Shared AGENTS.md instructions."), Visibility: firstNonEmpty(m.Spec.Visibility, cfg.Publish.Visibility, "private")})
		if err != nil {
			return fmt.Errorf("create package %s: %w", ref.PackagePath(), err)
		}
	}

	version, err := nextVersion(versions, bump, explicitVersion)
	if err != nil {
		return err
	}
	for _, existing := range versions {
		if existing.Version == version && existing.Lifecycle != "draft" {
			suggested, _ := nextVersion(versions, "patch", "")
			return fmt.Errorf("version %s already exists with lifecycle %s; try --version %s", version, existing.Lifecycle, suggested)
		}
	}
	m = applyGeneratedManifestFields(m, ref, version, cfg, visibilityOverride, channelOverride)
	if problems := validateAgentsManifest(m); len(problems) > 0 {
		return fmt.Errorf("manifest is invalid: %s", problems[0])
	}
	manifestBytes, err := manifestYAMLBytes(m)
	if err != nil {
		return err
	}
	agentsBytes, err := os.ReadFile(agentsMDPath)
	if err != nil {
		return err
	}

	if _, err := client.CreateDraft(ref.Org, ref.Namespace, ref.Name, CreateDraftRequest{Version: version, Visibility: m.Spec.Visibility}); err != nil {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Code != "VERSION_ALREADY_EXISTS" {
			return fmt.Errorf("create draft %s@%s: %w", ref.PackagePath(), version, err)
		}
		versionResp, lookupErr := client.GetVersion(ref.Org, ref.Namespace, ref.Name, version)
		if lookupErr != nil {
			return fmt.Errorf("lookup existing version %s@%s: %w", ref.PackagePath(), version, lookupErr)
		}
		if versionResp.Lifecycle != "draft" {
			return fmt.Errorf("version %s already exists with lifecycle %s", version, versionResp.Lifecycle)
		}
	}

	if _, err := client.UploadArtifact(ref.Org, ref.Namespace, ref.Name, version, manifestPath, manifestBytes, "application/yaml"); err != nil {
		return fmt.Errorf("upload %s: %w", manifestPath, err)
	}
	if _, err := client.UploadArtifact(ref.Org, ref.Namespace, ref.Name, version, agentsMDPath, agentsBytes, "text/markdown"); err != nil {
		return fmt.Errorf("upload %s: %w", agentsMDPath, err)
	}

	result := VersionResponse{Org: ref.Org, Namespace: ref.Namespace, Package: ref.Name, Version: version, Lifecycle: "draft", Visibility: m.Spec.Visibility}
	reviewURL := ""
	switch mode {
	case "draft":
		// Upload only.
	case "submit":
		if err := client.SubmitReview(ref.Org, ref.Namespace, ref.Name, version); err != nil {
			return fmt.Errorf("submit review: %w", err)
		}
		reviewURL = client.ServerURL + "/reviews"
	case "publish", "default":
		published, err := client.PublishVersion(ref.Org, ref.Namespace, ref.Name, version)
		if err != nil {
			var apiErr *APIError
			if mode == "default" && errors.As(err, &apiErr) && apiErr.Code == "APPROVAL_REQUIRED" {
				if submitErr := client.SubmitReview(ref.Org, ref.Namespace, ref.Name, version); submitErr != nil {
					return fmt.Errorf("submit review after approval required: %w", submitErr)
				}
				reviewURL = client.ServerURL + "/reviews"
				break
			}
			return fmt.Errorf("publish %s@%s: %w", ref.PackagePath(), version, err)
		}
		result = *published
	}

	if err := writeManifestYAML(manifestPath, m); err != nil {
		return err
	}
	_ = updateStateAfterPush(remoteName, ref.Selector, result, manifestBytes, agentsBytes)

	if jsonOutput {
		return outputJSON(map[string]string{"package": ref.PackagePath(), "version": version, "lifecycle": firstNonEmpty(result.Lifecycle, "review"), "digest": result.Digest, "channel": m.Spec.Channel, "visibility": m.Spec.Visibility, "reviewUrl": reviewURL, "installCommand": "trove install " + ref.PackagePath() + "@" + m.Spec.Channel})
	}
	if reviewURL != "" {
		fmt.Printf("Uploaded %s@%s\n", ref.PackagePath(), version)
		fmt.Println("Submitted for review:")
		fmt.Printf("  %s\n", reviewURL)
		return nil
	}
	if mode == "draft" {
		fmt.Printf("Uploaded draft %s@%s\n", ref.PackagePath(), version)
		return nil
	}
	fmt.Printf("Published %s@%s\n", ref.PackagePath(), version)
	fmt.Println("Install with:")
	fmt.Printf("  trove install %s@%s\n", ref.PackagePath(), m.Spec.Channel)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func updateStateAfterPush(remoteName, requestedSelector string, result VersionResponse, manifestBytes []byte, agentsBytes []byte) error {
	selector := requestedSelector
	if selector == "" {
		selector = "stable"
	}
	state := ProjectState{APIVersion: projectAPIVersion, Kind: projectStateKind, Source: ProjectStateSource{Remote: remoteName, RequestedSelector: selector, ResolvedVersion: result.Version, PackageDigest: result.Digest}, Files: map[string]StateFile{manifestPath: {Digest: computeDigest(manifestBytes)}, agentsMDPath: {Digest: computeDigest(agentsBytes)}}}
	return writeProjectState(projectStatePath, state)
}
