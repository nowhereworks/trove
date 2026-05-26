package cli

import (
	"fmt"
	"os"
	"strings"
)

func RunInit(args []string, jsonOutput bool) error {
	targetAgentsMD := hasFlag(args, "--agents-md")
	skipNext := false
	valueFlags := map[string]bool{"--remote": true, "--package": true, "--display-name": true, "--description": true}
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "agents-md" {
			targetAgentsMD = true
			continue
		}
		if valueFlags[arg] {
			skipNext = true
			continue
		}
		matchedValueFlag := false
		for flag := range valueFlags {
			if strings.HasPrefix(arg, flag+"=") {
				matchedValueFlag = true
				break
			}
		}
		if matchedValueFlag {
			continue
		}
		if arg != "--json" && arg != "--yes" && arg != "--force" && arg != "--agents-md" && arg != "--remote" && arg != "--package" && arg != "--display-name" && arg != "--description" && !strings.HasPrefix(arg, "--") {
			return fmt.Errorf("unsupported init target %q; expected agents-md", arg)
		}
	}
	if !targetAgentsMD {
		return fmt.Errorf("usage: trove init agents-md [--remote url-or-package-ref]")
	}

	force := hasFlag(args, "--force")
	remoteValue := flagValue(args, "--remote")
	packageValue := flagValue(args, "--package")
	displayName := flagValue(args, "--display-name")
	description := flagValue(args, "--description")

	// Check for existing trove.yaml before any work to avoid accidental overwrites.
	if _, err := os.Stat(manifestPath); err == nil && !force {
		return fmt.Errorf("%s already exists; use --force to overwrite", manifestPath)
	}

	existingConfig := readOptionalProjectConfig()
	var remoteSpec RemoteSpec
	hasRemote := false
	if remoteValue != "" {
		parsed, err := parseRemoteSpec(remoteValue, existingConfig)
		if err != nil {
			return err
		}
		remoteSpec = parsed
		hasRemote = true
	}

	var ref PackageRef
	hasPackage := false
	if hasRemote {
		ref = remoteSpec.Ref
		hasPackage = true
	} else if packageValue != "" {
		parsed, err := ParsePackageRefNoSelector(packageValue)
		if err != nil {
			return err
		}
		ref = parsed
		hasPackage = true
	} else if existingManifest, ok := readManifestIfExists(); ok && existingManifest.Metadata.Org != "" {
		ref = PackageRef{Org: existingManifest.Metadata.Org, Namespace: existingManifest.Metadata.Namespace, Name: existingManifest.Metadata.Name}
		hasPackage = true
	} else {
		cwd, _ := os.Getwd()
		ref = PackageRef{Org: "nwks", Namespace: "platform", Name: slugFromDir(cwd)}
		hasPackage = true
	}

	if _, err := os.Stat(agentsMDPath); os.IsNotExist(err) {
		starter := []byte("# AGENTS.md\n\nShared agent instructions for this repository.\n")
		if err := os.WriteFile(agentsMDPath, starter, 0644); err != nil {
			return fmt.Errorf("write %s: %w", agentsMDPath, err)
		}
	} else if err != nil {
		return err
	}

	if hasPackage && shouldWriteGenerated(manifestPath, force) {
		m := generatedAgentsManifest(ref, displayName, description)
		if err := writeManifestYAML(manifestPath, m); err != nil {
			return fmt.Errorf("write %s: %w", manifestPath, err)
		}
	}

	if shouldWriteGenerated(projectConfigPath, force) {
		cfg := ProjectConfig{APIVersion: projectAPIVersion, Kind: projectKind, ArtifactKind: agentsMDKind}
		if hasRemote {
			cfg = configWithRemote(remoteSpec)
		} else if packageValue != "" {
			if serverURL := serverURLForPackageInit(); serverURL != "" {
				cfg.DefaultRemote = "origin"
				cfg.Remotes = map[string]ProjectRemote{"origin": {ServerURL: serverURL, Package: ref.PackagePath()}}
			}
		}
		if err := writeProjectConfig(projectConfigPath, cfg); err != nil {
			return fmt.Errorf("write %s: %w", projectConfigPath, err)
		}
	}

	if jsonOutput {
		out := map[string]string{"artifactKind": agentsMDKind, "artifactPath": agentsMDPath, "manifestPath": manifestPath, "configPath": projectConfigPath}
		if hasRemote {
			out["remote"] = "origin"
			out["serverUrl"] = remoteSpec.ServerURL
			out["package"] = remoteSpec.Package
		} else if hasPackage {
			out["package"] = ref.PackagePath()
		}
		return outputJSON(out)
	}

	fmt.Println("Initialized AGENTS.md package worktree")
	fmt.Println("Artifact: AGENTS.md")
	fmt.Println("Manifest: trove.yaml")
	fmt.Println("Config: .trove/config.yaml")
	if hasRemote {
		fmt.Printf("Remote: origin -> %s/%s\n", remoteSpec.ServerURL, remoteSpec.Package)
	}
	return nil
}

func serverURLForPackageInit() string {
	if serverURL := os.Getenv("TROVE_SERVER_URL"); serverURL != "" {
		return serverURL
	}
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return ""
	}
	if cfg.DefaultRemote != "" {
		if remote, ok := cfg.Remotes[cfg.DefaultRemote]; ok {
			return remote.ServerURL
		}
	}
	for _, remote := range cfg.Remotes {
		if remote.ServerURL != "" {
			return remote.ServerURL
		}
	}
	return ""
}
