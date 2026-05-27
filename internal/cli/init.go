package cli

import (
	"fmt"
	"os"
	"strings"

	"trove/internal/manifest"
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

	if _, err := os.Stat(manifestPath); err == nil && !force {
		return fmt.Errorf("%s already exists; use --force to overwrite", manifestPath)
	}

	existing := readOptionalTrovefile()
	var remoteSpec RemoteSpec
	hasRemote := false
	if remoteValue != "" {
		parsed, err := parseRemoteSpec(remoteValue, existing)
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
	} else if existingManifest, ok := readTrovefileIfExists(); ok && existingManifest.Metadata.Org != "" {
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
		if hasRemote {
			m.Local = &manifest.Local{
				DefaultRemote: "origin",
				Remotes: map[string]manifest.ProjectRemote{
					"origin": {ServerURL: remoteSpec.ServerURL, Package: remoteSpec.Package},
				},
			}
		} else if packageValue != "" {
			if serverURL := serverURLForPackageInit(); serverURL != "" {
				m.Local = &manifest.Local{
					DefaultRemote: "origin",
					Remotes: map[string]manifest.ProjectRemote{
						"origin": {ServerURL: serverURL, Package: ref.PackagePath()},
					},
				}
			}
		}
		if err := writeTrovefile(manifestPath, m); err != nil {
			return fmt.Errorf("write %s: %w", manifestPath, err)
		}
	}

	if jsonOutput {
		out := map[string]string{"artifactKind": agentsMDKind, "artifactPath": agentsMDPath, "manifestPath": manifestPath}
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
	fmt.Printf("Manifest: %s\n", manifestPath)
	if hasRemote {
		fmt.Printf("Remote: origin -> %s/%s\n", remoteSpec.ServerURL, remoteSpec.Package)
	}
	return nil
}

func serverURLForPackageInit() string {
	if serverURL := os.Getenv("TROVE_SERVER_URL"); serverURL != "" {
		return serverURL
	}
	m, err := loadTrovefile(manifestPath)
	if err != nil {
		return ""
	}
	if m.Local != nil && m.Local.DefaultRemote != "" {
		if remote, ok := m.Local.Remotes[m.Local.DefaultRemote]; ok {
			return remote.ServerURL
		}
	}
	if m.Local != nil {
		for _, remote := range m.Local.Remotes {
			if remote.ServerURL != "" {
				return remote.ServerURL
			}
		}
	}
	return ""
}
