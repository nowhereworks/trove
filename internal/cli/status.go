package cli

import (
	"fmt"
	"os"

	"trove/internal/manifest"
)

type statusOutput struct {
	ArtifactKind            string        `json:"artifactKind"`
	ArtifactPath            string        `json:"artifactPath"`
	Remote                  *statusRemote `json:"remote,omitempty"`
	CurrentPublishedVersion string        `json:"currentPublishedVersion"`
	NextVersion             string        `json:"nextVersion"`
	Visibility              string        `json:"visibility"`
	ReviewPolicy            string        `json:"reviewPolicy"`
	LocalState              string        `json:"localState"`
	Problems                []string      `json:"problems"`
}

type statusRemote struct {
	Name      string `json:"name"`
	ServerURL string `json:"serverUrl"`
	Package   string `json:"package"`
}

func RunStatus(args []string, jsonOutput bool) error {
	_ = args
	out := statusOutput{ArtifactKind: agentsMDKind, ArtifactPath: agentsMDPath, ReviewPolicy: "unknown", LocalState: "ready", Problems: []string{}}

	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		out.Problems = append(out.Problems, "missing .trove/config.yaml; run 'trove init agents-md'")
	} else {
		out.Problems = append(out.Problems, validateProjectConfig(cfg)...)
		out.Visibility = cfg.Publish.Visibility
		if name, remote, err := remoteForConfig(cfg, ""); err == nil {
			out.Remote = &statusRemote{Name: name, ServerURL: remote.ServerURL, Package: remote.Package}
			versions, _, err := packageVersionsForRemote(remote)
			if err == nil {
				out.NextVersion, _ = nextVersion(versions, "patch", "")
				out.CurrentPublishedVersion = latestPublishedVersion(versions)
				refreshReviewPolicy(&out, remote, out.NextVersion)
			} else {
				out.NextVersion = "1.0.0"
				out.Problems = append(out.Problems, "remote package lookup failed: "+err.Error())
			}
		} else {
			out.Problems = append(out.Problems, err.Error())
			out.NextVersion = "1.0.0"
		}
	}

	m, err := loadManifestYAML(manifestPath)
	if err != nil {
		out.Problems = append(out.Problems, "missing or invalid trove.yaml: "+err.Error())
	} else {
		out.Problems = append(out.Problems, validateAgentsManifest(m)...)
		if out.Visibility == "" {
			out.Visibility = m.Spec.Visibility
		}
		if out.Remote != nil && m.Metadata.Org+"/"+m.Metadata.Namespace+"/"+m.Metadata.Name != out.Remote.Package {
			out.Problems = append(out.Problems, "manifest package does not match configured remote")
		}
	}

	if _, err := os.Stat(agentsMDPath); err != nil {
		out.Problems = append(out.Problems, "missing AGENTS.md")
	}
	if os.Getenv("TROVE_TOKEN") == "" {
		out.Problems = append(out.Problems, "missing TROVE_TOKEN for publishing")
	}

	if len(out.Problems) > 0 {
		out.LocalState = "blocked"
	}
	if out.Visibility == "" {
		out.Visibility = "private"
	}
	if out.NextVersion == "" {
		out.NextVersion = "1.0.0"
	}
	if out.ReviewPolicy == "unknown" {
		out.ReviewPolicy = "requiresApproval"
	}

	if jsonOutput {
		if err := outputJSON(out); err != nil {
			return err
		}
	} else {
		printStatus(out)
	}
	if len(out.Problems) > 0 {
		return fmt.Errorf("local state is not publishable")
	}
	return nil
}

func refreshReviewPolicy(out *statusOutput, remote ProjectRemote, version string) {
	if version == "" || os.Getenv("TROVE_TOKEN") == "" {
		return
	}
	ref, err := ParsePackageRefNoSelector(remote.Package)
	if err != nil {
		return
	}
	status, err := NewClientForServer(remote.ServerURL).ApprovalStatus(ref.Org, ref.Namespace, ref.Name, version)
	if err != nil {
		return
	}
	if status.RequiredCount == 0 {
		out.ReviewPolicy = "noApprovalRequired"
		return
	}
	if status.HasEnoughApprovals {
		out.ReviewPolicy = "approved"
		return
	}
	out.ReviewPolicy = "requiresApproval"
}

func latestPublishedVersion(versions []PackageVersion) string {
	var latest *SemVer
	latestRaw := ""
	for _, version := range versions {
		if version.Lifecycle != "published" {
			continue
		}
		parsed, err := ParseSemVer(version.Version)
		if err != nil {
			continue
		}
		if latest == nil || semverLess(*latest, parsed) {
			latest = &parsed
			latestRaw = parsed.String()
		}
	}
	return latestRaw
}

func printStatus(out statusOutput) {
	fmt.Printf("Artifact: %s\n", out.ArtifactPath)
	if out.Remote != nil {
		fmt.Printf("Remote: %s -> %s/%s\n", out.Remote.Name, out.Remote.ServerURL, out.Remote.Package)
	} else {
		fmt.Println("Remote: missing")
	}
	if out.CurrentPublishedVersion == "" {
		fmt.Println("Current published version: none")
	} else {
		fmt.Printf("Current published version: %s\n", out.CurrentPublishedVersion)
	}
	fmt.Printf("Next version: %s\n", out.NextVersion)
	fmt.Printf("Visibility: %s\n", out.Visibility)
	fmt.Printf("Review policy: %s\n", out.ReviewPolicy)
	fmt.Printf("Local state: %s\n", out.LocalState)
	for _, problem := range out.Problems {
		fmt.Printf("Problem: %s\n", problem)
	}
}

func applyGeneratedManifestFields(m manifest.Manifest, ref PackageRef, version string, cfg ProjectConfig, visibilityOverride string) manifest.Manifest {
	m.Metadata.Org = ref.Org
	m.Metadata.Namespace = ref.Namespace
	m.Metadata.Name = ref.Name
	if m.Metadata.DisplayName == "" {
		m.Metadata.DisplayName = titleFromSlug(ref.Name)
	}
	if m.Metadata.Description == "" {
		m.Metadata.Description = "Shared AGENTS.md instructions."
	}
	m.Spec.Version = version
	m.Spec.Lifecycle = "draft"
	if visibilityOverride != "" {
		m.Spec.Visibility = visibilityOverride
	} else if cfg.Publish.Visibility != "" {
		m.Spec.Visibility = cfg.Publish.Visibility
	} else if m.Spec.Visibility == "" {
		m.Spec.Visibility = "private"
	}
	m.Spec.Artifacts = agentsArtifacts()
	return m
}
