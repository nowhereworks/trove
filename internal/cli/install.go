package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"trove/internal/lockfile"
	"trove/internal/manifest"
)

func RunInstall(args []string, outputDir string, optional bool, overwrite bool, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove install <org/namespace/package@selector>")
	}

	ref, err := ParsePackageRef(args[0])
	if err != nil {
		return err
	}

	if outputDir == "" {
		outputDir = "."
	}

	client := NewClient()
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

	artifactPins := make([]lockfile.ArtifactPin, 0, len(m.Spec.Artifacts))
	for _, artifact := range m.Spec.Artifacts {
		if !artifact.Required && !optional {
			continue
		}

		targetPath := artifact.TargetPath
		if targetPath == "" {
			targetPath = artifact.Path
		}

		fullPath := filepath.Join(outputDir, targetPath)

		if !overwrite {
			if _, err := os.Stat(fullPath); err == nil {
				return fmt.Errorf("target file %s already exists; use --overwrite to replace", targetPath)
			}
		}

		artifactBytes, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, artifact.Path)
		if err != nil {
			return fmt.Errorf("fetch artifact %s: %w", artifact.Path, err)
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", targetPath, err)
		}

		if err := os.WriteFile(fullPath, artifactBytes, 0644); err != nil {
			return fmt.Errorf("write artifact %s: %w", targetPath, err)
		}

		digest := computeDigest(artifactBytes)
		artifactPins = append(artifactPins, lockfile.ArtifactPin{
			Source: artifact.Path,
			Target: targetPath,
			Digest: digest,
		})
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"package":  ref.PackagePath(),
			"version":  result.ResolvedVersion,
			"digest":   result.Digest,
			"artifacts": len(artifactPins),
		})
	}

	fmt.Printf("Installed %s/%s/%s@%s (%d artifacts)\n", ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, len(artifactPins))
	for _, pin := range artifactPins {
		fmt.Printf("  %s -> %s\n", pin.Source, pin.Target)
	}

	lockPath := filepath.Join(outputDir, lockfile.DefaultLockFile)
	var lock *lockfile.LockFile
	if existing, err := lockfile.ParseFile(lockPath); err == nil {
		lock = existing
	} else {
		lock = lockfile.New("trove", "0.1.0", lockfile.ProjectInfo{
			Org:  ref.Org,
			Name: ref.Name,
		})
	}

	lock.AddInstall(lockfile.InstallEntry{
		Package:           ref.PackagePath(),
		RequestedSelector: ref.Selector,
		Version:           result.ResolvedVersion,
		Digest:            result.Digest,
		InstalledAt:       time.Now().UTC(),
		Artifacts:         artifactPins,
	})

	if err := lock.WriteFile(lockPath); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}

	return nil
}

func computeDigest(data []byte) string {
	h := sha256Sum(data)
	return "sha256:" + h
}
