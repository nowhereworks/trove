package cli

import (
	"errors"
	"fmt"
	"net/url"
	"os"
)

func RunPush(args []string, jsonOutput bool) error {
	remoteNameFlag := flagValue(args, "--remote")
	visibilityOverride := flagValue(args, "--visibility")
	explicitVersion := flagValue(args, "--version")
	force := hasFlag(args, "--force")
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
	}

	version, err := nextVersion(versions, bump, explicitVersion)
	if err != nil {
		return err
	}
	for _, existing := range versions {
		if existing.Version == version && existing.Lifecycle != "draft" {
			suggested, _ := nextVersion(versions, "patch", "")
			if force {
				return fmt.Errorf("version %s already exists with immutable lifecycle %s; try --version %s", version, existing.Lifecycle, suggested)
			}
			return fmt.Errorf("version %s already exists with lifecycle %s; try --version %s", version, existing.Lifecycle, suggested)
		}
	}
	m = applyGeneratedManifestFields(m, ref, version, cfg, visibilityOverride)
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
		switch versionResp.Lifecycle {
		case "draft":
			// Continue below; artifact uploads overwrite draft content.
		case "review":
			if !force {
				return fmt.Errorf("version %s is already in review; use --force to reset it to draft or choose a new version", version)
			}
			if _, resetErr := client.ResetDraft(ref.Org, ref.Namespace, ref.Name, version); resetErr != nil {
				return fmt.Errorf("reset review version %s@%s: %w", ref.PackagePath(), version, resetErr)
			}
		case "published", "deprecated", "yanked":
			suggested, _ := nextVersion(versions, "patch", "")
			return fmt.Errorf("version %s already exists with immutable lifecycle %s; try --version %s", version, versionResp.Lifecycle, suggested)
		default:
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
		result.Lifecycle = "review"
		reviewURL = reviewPageURL(client.ServerURL, ref, version)
	case "publish", "default":
		published, err := client.PublishVersion(ref.Org, ref.Namespace, ref.Name, version)
		if err != nil {
			var apiErr *APIError
			if mode == "default" && errors.As(err, &apiErr) && apiErr.Code == "APPROVAL_REQUIRED" {
				if submitErr := client.SubmitReview(ref.Org, ref.Namespace, ref.Name, version); submitErr != nil {
					return fmt.Errorf("submit review after approval required: %w", submitErr)
				}
				result.Lifecycle = "review"
				reviewURL = reviewPageURL(client.ServerURL, ref, version)
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
		return outputJSON(map[string]string{"package": ref.PackagePath(), "version": version, "lifecycle": firstNonEmpty(result.Lifecycle, "review"), "digest": result.Digest, "visibility": m.Spec.Visibility, "reviewUrl": reviewURL, "installCommand": "trove install " + ref.PackagePath() + "@latest"})
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
	fmt.Printf("  trove install %s@latest\n", ref.PackagePath())
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
		selector = "latest"
	}
	manifestDigest := computeDigest(manifestBytes)
	state := ProjectState{APIVersion: projectAPIVersion, Kind: projectStateKind, Source: ProjectStateSource{Remote: remoteName, RequestedSelector: selector, ResolvedVersion: result.Version, PackageDigest: result.Digest, ManifestDigest: manifestDigest}, Files: map[string]StateFile{manifestPath: {Digest: manifestDigest}, agentsMDPath: {Digest: computeDigest(agentsBytes)}}}
	return writeProjectState(projectStatePath, state)
}

func reviewPageURL(serverURL string, ref PackageRef, version string) string {
	values := url.Values{}
	values.Set("package", ref.PackagePath())
	values.Set("version", version)
	return serverURL + "/reviews?" + values.Encode()
}
