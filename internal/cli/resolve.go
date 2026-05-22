package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

func RunResolve(args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove resolve <org/namespace/package@selector>")
	}

	ref, err := ParsePackageRef(args[0])
	if err != nil {
		return err
	}

	client := NewClient()
	result, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, ref.Selector)
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("Package: %s/%s/%s\n", result.Org, result.Namespace, result.Package)
	fmt.Printf("Selector: %s\n", ref.Selector)
	fmt.Printf("Resolved version: %s\n", result.ResolvedVersion)
	fmt.Printf("Digest: %s\n", result.Digest)
	fmt.Printf("Manifest URL: %s\n", result.ManifestURL)
	fmt.Printf("Archive URL: %s\n", result.ArchiveURL)
	return nil
}

func RunFetch(args []string, outputDir string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove fetch <org/namespace/package@selector> [--output dir]")
	}

	ref, err := ParsePackageRef(args[0])
	if err != nil {
		return err
	}

	if outputDir == "" {
		outputDir = ref.Name
	}

	client := NewClient()
	result, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, ref.Selector)
	if err != nil {
		return err
	}

	archiveBytes, err := client.GetArchive(ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, "tar.gz")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := extractTarGz(archiveBytes, outputDir); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]string{
			"package":  ref.PackagePath(),
			"version":  result.ResolvedVersion,
			"digest":   result.Digest,
			"output":   outputDir,
		})
	}

	fmt.Printf("Fetched %s/%s/%s@%s to %s\n", ref.Org, ref.Namespace, ref.Name, result.ResolvedVersion, outputDir)
	return nil
}

func extractTarGz(data []byte, dest string) error {
	return extractArchive(data, dest, "tar.gz")
}
