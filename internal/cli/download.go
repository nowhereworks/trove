package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunDownload(args []string, outputPath string, overwrite bool, metadataOnly bool, jsonOutput bool) error {
	if len(args) == 1 {
		if skillName, ok := parseCoreSkillDownloadPath(args[0]); ok {
			return runCoreSkillDownload(args[0], skillName, outputPath, overwrite, metadataOnly, jsonOutput)
		}
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: trove download <org/namespace/package[@selector]> <artifact-path> | core/skills/<name>/SKILL.md")
	}
	if jsonOutput && outputPath == "" && !metadataOnly {
		return fmt.Errorf("--json requires --output or --metadata-only because stdout is reserved for JSON")
	}

	ref, err := ParsePackageRef(args[0])
	if err != nil {
		return err
	}
	artifactPath := args[1]
	if artifactPath == "" || filepath.IsAbs(artifactPath) || filepath.Clean(artifactPath) != artifactPath || bytes.Contains([]byte(artifactPath), []byte("..")) || bytes.Contains([]byte(artifactPath), []byte("@")) {
		return fmt.Errorf("invalid artifact path %q", artifactPath)
	}

	client := NewClient()
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
