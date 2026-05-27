package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"trove/internal/manifest"
)

func RunDownload(args []string, outputPath string, overwrite bool, metadataOnly bool, jsonOutput bool) error {
	if len(args) == 1 {
		if skillName, ok := parseCoreSkillDownloadPath(args[0]); ok {
			return runCoreSkillDownload(args[0], skillName, outputPath, overwrite, metadataOnly, jsonOutput)
		}
		return runPackageDownload(args[0], outputPath, overwrite, metadataOnly, jsonOutput)
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: trove download [server-url/]org/namespace/package[@selector] [artifact-path] | core/skills/<name>/SKILL.md")
	}
	if jsonOutput && outputPath == "" && !metadataOnly {
		return fmt.Errorf("--json requires --output or --metadata-only because stdout is reserved for JSON")
	}

	ref, client, err := parseDownloadSource(args[0])
	if err != nil {
		return err
	}
	artifactPath := args[1]
	if !validDownloadArtifactPath(artifactPath) {
		return fmt.Errorf("invalid artifact path %q", artifactPath)
	}

	result, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, ref.Selector)
	if err != nil {
		return err
	}

	metadata := map[string]string{
		"package":       ref.PackagePath(),
		"selector":      ref.Selector,
		"version":       result.ResolvedVersion,
		"packageDigest": result.Digest,
		"artifactPath":  artifactPath,
	}
	if outputPath != "" {
		metadata["output"] = outputPath
	}

	if metadataOnly {
		if jsonOutput {
			return writeJSONToStdout(metadata)
		}
		fmt.Printf("Package: %s\n", ref.PackagePath())
		fmt.Printf("Selector: %s\n", ref.Selector)
		fmt.Printf("Resolved version: %s\n", result.ResolvedVersion)
		fmt.Printf("Artifact: %s\n", artifactPath)
		return nil
	}

	artifactBytes, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, artifactPath)
	if err != nil {
		return fmt.Errorf("download artifact %s: %w", artifactPath, err)
	}

	if outputPath == "" {
		_, err := os.Stdout.Write(artifactBytes)
		return err
	}

	changed, err := writeDownloadedBytes(artifactBytes, outputPath, overwrite)
	if err != nil {
		return err
	}
	metadata["changed"] = changed

	if jsonOutput {
		return writeJSONToStdout(metadata)
	}
	if changed == "false" {
		fmt.Printf("Downloaded %s@%s %s to %s (unchanged)\n", ref.PackagePath(), result.ResolvedVersion, artifactPath, outputPath)
		return nil
	}
	fmt.Printf("Downloaded %s@%s %s to %s\n", ref.PackagePath(), result.ResolvedVersion, artifactPath, outputPath)
	return nil
}

func runPackageDownload(source string, outputDir string, overwrite bool, metadataOnly bool, jsonOutput bool) error {
	ref, client, err := parseDownloadSource(source)
	if err != nil {
		return err
	}
	if outputDir == "" {
		outputDir = "."
	}

	result, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, ref.Selector)
	if err != nil {
		return err
	}
	manifestResp, err := client.GetManifest(ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion)
	if err != nil {
		return err
	}
	var m manifest.Manifest
	if err := json.Unmarshal(manifestResp.Raw, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	metadata := map[string]string{
		"package":       ref.PackagePath(),
		"selector":      ref.Selector,
		"version":       result.ResolvedVersion,
		"packageDigest": result.Digest,
		"output":        outputDir,
	}
	if metadataOnly {
		metadata["artifacts"] = fmt.Sprint(countDownloadableArtifacts(m.Spec.Artifacts))
		if jsonOutput {
			return writeJSONToStdout(metadata)
		}
		fmt.Printf("Package: %s\n", ref.PackagePath())
		fmt.Printf("Selector: %s\n", ref.Selector)
		fmt.Printf("Resolved version: %s\n", result.ResolvedVersion)
		fmt.Printf("Artifacts: %s\n", metadata["artifacts"])
		return nil
	}

	changedCount := 0
	artifactCount := 0
	for _, artifact := range m.Spec.Artifacts {
		if isProjectMetadataPath(artifact.Path) {
			continue
		}
		targetPath := artifact.TargetPath
		if targetPath == "" {
			targetPath = artifact.Path
		}
		if isProjectMetadataPath(targetPath) {
			continue
		}
		if !validDownloadArtifactPath(artifact.Path) || !validDownloadArtifactPath(targetPath) {
			return fmt.Errorf("invalid artifact path %q", artifact.Path)
		}
		artifactBytes, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, artifact.Path)
		if err != nil {
			return fmt.Errorf("download artifact %s: %w", artifact.Path, err)
		}
		changed, err := writeDownloadedBytes(artifactBytes, filepath.Join(outputDir, targetPath), overwrite)
		if err != nil {
			return err
		}
		artifactCount++
		if changed == "true" {
			changedCount++
		}
	}
	metadata["artifacts"] = fmt.Sprint(artifactCount)
	metadata["changed"] = fmt.Sprint(changedCount)

	if jsonOutput {
		return writeJSONToStdout(metadata)
	}
	fmt.Printf("Downloaded %s@%s to %s (%d artifacts, %d changed)\n", ref.PackagePath(), result.ResolvedVersion, outputDir, artifactCount, changedCount)
	return nil
}

func runCoreSkillDownload(corePath, skillName, outputPath string, overwrite bool, metadataOnly bool, jsonOutput bool) error {
	if jsonOutput && outputPath == "" && !metadataOnly {
		return fmt.Errorf("--json requires --output or --metadata-only because stdout is reserved for JSON")
	}

	skillBytes, err := NewClient().GetCoreSkill(skillName)
	if err != nil {
		return fmt.Errorf("download core skill %s: %w", skillName, err)
	}

	metadata := map[string]string{
		"source":       "core",
		"coreSkill":    skillName,
		"artifactPath": corePath,
	}
	if outputPath != "" {
		metadata["output"] = outputPath
	}

	if metadataOnly {
		if jsonOutput {
			return writeJSONToStdout(metadata)
		}
		fmt.Printf("Source: core\n")
		fmt.Printf("Core skill: %s\n", skillName)
		fmt.Printf("Artifact: %s\n", corePath)
		return nil
	}

	if outputPath == "" {
		_, err := os.Stdout.Write(skillBytes)
		return err
	}

	changed, err := writeDownloadedBytes(skillBytes, outputPath, overwrite)
	if err != nil {
		return err
	}
	metadata["changed"] = changed

	if jsonOutput {
		return writeJSONToStdout(metadata)
	}
	if changed == "false" {
		fmt.Printf("Downloaded core skill %s to %s (unchanged)\n", skillName, outputPath)
		return nil
	}
	fmt.Printf("Downloaded core skill %s to %s\n", skillName, outputPath)
	return nil
}

func parseCoreSkillDownloadPath(path string) (string, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[0] != "core" || parts[1] != "skills" || parts[3] != "SKILL.md" {
		return "", false
	}
	name := parts[2]
	if name == "" || filepath.IsAbs(path) || filepath.Clean(path) != path || strings.Contains(path, "..") || strings.Contains(path, "@") {
		return "", false
	}
	return name, true
}

func parseDownloadSource(source string) (PackageRef, *Client, error) {
	if u, err := url.Parse(source); err == nil && u.Scheme != "" && u.Host != "" {
		ref, err := ParsePackageRef(strings.TrimPrefix(u.EscapedPath(), "/"))
		if err != nil {
			return PackageRef{}, nil, err
		}
		serverURL := strings.TrimRight(u.Scheme+"://"+u.Host, "/")
		return ref, NewClientForServer(serverURL), nil
	}
	ref, err := ParsePackageRef(source)
	if err != nil {
		return PackageRef{}, nil, err
	}
	return ref, NewClient(), nil
}

func validDownloadArtifactPath(path string) bool {
	return path != "" && !filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.Contains(path, "..") && !strings.Contains(path, "@") && !isProjectMetadataPath(path)
}

func isProjectMetadataPath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == manifestPath || clean == ".trove.lock.yaml" || clean == ".trove" || strings.HasPrefix(clean, ".trove/")
}

func countDownloadableArtifacts(artifacts []manifest.Artifact) int {
	count := 0
	for _, artifact := range artifacts {
		targetPath := artifact.TargetPath
		if targetPath == "" {
			targetPath = artifact.Path
		}
		if !isProjectMetadataPath(artifact.Path) && !isProjectMetadataPath(targetPath) {
			count++
		}
	}
	return count
}

func writeDownloadedBytes(data []byte, outputPath string, overwrite bool) (string, error) {
	changed := "true"
	if existing, err := os.ReadFile(outputPath); err == nil {
		if bytes.Equal(existing, data) {
			changed = "false"
		} else if !overwrite {
			return "", fmt.Errorf("output file %s already exists and differs; use --overwrite to replace", outputPath)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read output file %s: %w", outputPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	if changed == "true" {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return "", fmt.Errorf("write artifact %s: %w", outputPath, err)
		}
	}
	return changed, nil
}

func writeJSONToStdout(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
