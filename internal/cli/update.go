package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"trove/internal/lockfile"
)

type UpdateResult struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Updated        bool   `json:"updated"`
}

func RunUpdate(args []string, apply bool, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	lockPath := filepath.Join(cwd, lockfile.DefaultLockFile)
	lock, err := lockfile.ParseFile(lockPath)
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}

	client := NewClient()
	results := make([]UpdateResult, 0, len(lock.Installs))
	updated := false

	for i, install := range lock.Installs {
		resp, err := client.CheckUpdate(UpdateCheckRequest{
			Package:        install.Package,
			CurrentVersion: install.Version,
			CurrentDigest:  install.Digest,
		})
		if err != nil {
			if jsonOutput {
				return fmt.Errorf("check update for %s: %w", install.Package, err)
			}
			fmt.Fprintf(os.Stderr, "ERROR checking %s: %v\n", install.Package, err)
			continue
		}

		result := UpdateResult{
			Package:        install.Package,
			CurrentVersion: install.Version,
			LatestVersion:  resp.LatestVersion,
		}

		if resp.UpdateAvailable && apply {
			ref, err := ParsePackageRef(install.Package + "@" + install.RequestedSelector)
			if err != nil {
				if jsonOutput {
					return fmt.Errorf("parse package ref for %s: %w", install.Package, err)
				}
				fmt.Fprintf(os.Stderr, "ERROR parsing %s: %v\n", install.Package, err)
				continue
			}

			manifestResp, err := client.GetManifest(ref.Org, ref.Namespace, ref.Name, resp.LatestVersion)
			if err != nil {
				if jsonOutput {
					return fmt.Errorf("fetch manifest for %s@%s: %w", install.Package, resp.LatestVersion, err)
				}
				fmt.Fprintf(os.Stderr, "ERROR fetching manifest for %s@%s: %v\n", install.Package, resp.LatestVersion, err)
				continue
			}

			var m struct {
				Spec struct {
					Artifacts []struct {
						Path       string `json:"path"`
						Type       string `json:"type"`
						Required   bool   `json:"required"`
						TargetPath string `json:"targetPath"`
					} `json:"artifacts"`
				} `json:"spec"`
			}
			if err := json.Unmarshal(manifestResp.Raw, &m); err != nil {
				if jsonOutput {
					return fmt.Errorf("parse manifest for %s: %w", install.Package, err)
				}
				fmt.Fprintf(os.Stderr, "ERROR parsing manifest for %s: %v\n", install.Package, err)
				continue
			}

			artifactPins := make([]lockfile.ArtifactPin, 0, len(m.Spec.Artifacts))
			for _, artifact := range m.Spec.Artifacts {
				if !artifact.Required {
					continue
				}

				targetPath := artifact.TargetPath
				if targetPath == "" {
					targetPath = artifact.Path
				}

				fullPath := filepath.Join(cwd, targetPath)
				artifactBytes, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, resp.LatestVersion, artifact.Path)
				if err != nil {
					if jsonOutput {
						return fmt.Errorf("fetch artifact %s: %w", artifact.Path, err)
					}
					fmt.Fprintf(os.Stderr, "ERROR fetching artifact %s: %v\n", artifact.Path, err)
					continue
				}

				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					if jsonOutput {
						return fmt.Errorf("create directory for %s: %w", targetPath, err)
					}
					fmt.Fprintf(os.Stderr, "ERROR creating directory for %s: %v\n", targetPath, err)
					continue
				}

				if err := os.WriteFile(fullPath, artifactBytes, 0644); err != nil {
					if jsonOutput {
						return fmt.Errorf("write artifact %s: %w", targetPath, err)
					}
					fmt.Fprintf(os.Stderr, "ERROR writing artifact %s: %v\n", targetPath, err)
					continue
				}

				digest := computeDigest(artifactBytes)
				artifactPins = append(artifactPins, lockfile.ArtifactPin{
					Source: artifact.Path,
					Target: targetPath,
					Digest: digest,
				})
			}

			lock.Installs[i].Version = resp.LatestVersion
			lock.Installs[i].Digest = resp.LatestDigest
			lock.Installs[i].InstalledAt = time.Now().UTC()
			lock.Installs[i].Artifacts = artifactPins
			updated = true
			result.Updated = true
		}

		results = append(results, result)
	}

	if apply && updated {
		if err := lock.WriteFile(lockPath); err != nil {
			return fmt.Errorf("write lockfile: %w", err)
		}
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"results": results,
			"applied": apply && updated,
		})
	}

	if !apply {
		fmt.Println("Dry run. Use --apply to update lockfile and artifacts.")
	}

	for _, r := range results {
		if r.Updated {
			fmt.Printf("%s: updated %s -> %s\n", r.Package, r.CurrentVersion, r.LatestVersion)
		} else if r.CurrentVersion == r.LatestVersion {
			fmt.Printf("%s: up to date (%s)\n", r.Package, r.CurrentVersion)
		} else {
			fmt.Printf("%s: update available %s -> %s\n", r.Package, r.CurrentVersion, r.LatestVersion)
		}
	}

	if apply && updated {
		fmt.Printf("Lockfile updated: %s\n", lockPath)
	}

	return nil
}
