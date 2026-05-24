package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"trove/internal/lockfile"
)

func sha256Sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

type CheckResult struct {
	Package          string `json:"package"`
	CurrentVersion   string `json:"currentVersion"`
	LatestVersion    string `json:"latestVersion"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	Compatibility    string `json:"compatibility"`
	RequiresApproval bool   `json:"requiresManualApproval"`
	ChangelogURL     string `json:"changelogUrl,omitempty"`
}

func RunCheck(args []string, jsonOutput bool) error {
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
	results := make([]CheckResult, 0, len(lock.Installs))
	hasIncompatible := false

	for _, install := range lock.Installs {
		ref, err := ParsePackageRef(install.Package + "@" + install.RequestedSelector)
		if err != nil {
			ref = PackageRef{
				Org:       "",
				Namespace: "",
				Name:      "",
				Selector:  install.RequestedSelector,
			}
			if idx := findPackageParts(install.Package); idx != nil {
				ref.Org = idx[0]
				ref.Namespace = idx[1]
				ref.Name = idx[2]
			}
		}

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

		result := CheckResult{
			Package:          install.Package,
			CurrentVersion:   install.Version,
			LatestVersion:    resp.LatestVersion,
			UpdateAvailable:  resp.UpdateAvailable,
			Compatibility:    resp.Compatibility,
			RequiresApproval: resp.RequiresManualApproval,
			ChangelogURL:     resp.ChangelogURL,
		}
		results = append(results, result)

		if resp.Compatibility == "incompatible" {
			hasIncompatible = true
		}
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"results":            results,
			"hasIncompatible":    hasIncompatible,
			"hasUpdateAvailable": hasUpdate(results),
		})
	}

	if hasIncompatible {
		fmt.Fprintln(os.Stderr, "WARNING: incompatible package versions detected")
	}

	for _, r := range results {
		status := "up to date"
		if r.UpdateAvailable {
			status = "update available: " + r.LatestVersion
		}
		compat := ""
		if r.Compatibility != "" && r.Compatibility != "unknown" {
			compat = " [" + r.Compatibility + "]"
		}
		fmt.Printf("%s: %s%s\n", r.Package, status, compat)
	}

	if hasIncompatible {
		return fmt.Errorf("incompatible versions found")
	}

	return nil
}

func hasUpdate(results []CheckResult) bool {
	for _, r := range results {
		if r.UpdateAvailable {
			return true
		}
	}
	return false
}

func findPackageParts(pkg string) []string {
	for i := len(pkg) - 1; i >= 0; i-- {
		if pkg[i] == '/' {
			rest := pkg[:i]
			for j := i - 1; j >= 0; j-- {
				if rest[j] == '/' {
					return []string{rest[:j], rest[j+1 : i], pkg[i+1:]}
				}
			}
		}
	}
	return nil
}
