package cli

import "fmt"

func RunRemote(args []string, jsonOutput bool, force bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove remote add|list|remove")
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: trove remote add <name> <url-or-package-ref>")
		}
		return remoteAdd(args[1], args[2], force)
	case "list":
		return remoteList(jsonOutput)
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: trove remote remove <name>")
		}
		return remoteRemove(args[1])
	default:
		return fmt.Errorf("unknown remote subcommand: %s", args[0])
	}
}

func remoteAdd(name string, value string, force bool) error {
	cfg := readOptionalProjectConfig()
	if cfg.APIVersion == "" {
		cfg = ProjectConfig{APIVersion: projectAPIVersion, Kind: projectKind, ArtifactKind: agentsMDKind}
	}
	spec, err := parseRemoteSpec(value, cfg)
	if err != nil {
		return err
	}
	if cfg.Remotes == nil {
		cfg.Remotes = map[string]ProjectRemote{}
	}
	if _, exists := cfg.Remotes[name]; exists && !force {
		return fmt.Errorf("remote %q already exists; use --force to replace", name)
	}
	cfg.Remotes[name] = ProjectRemote{ServerURL: spec.ServerURL, Package: spec.Package}
	if cfg.DefaultRemote == "" {
		cfg.DefaultRemote = name
	}
	if err := writeProjectConfig(projectConfigPath, cfg); err != nil {
		return err
	}
	if m, ok := readManifestIfExists(); ok {
		if m.Metadata.Org == "" || m.Metadata.Org == "nwks" {
			m.Metadata.Org = spec.Ref.Org
			m.Metadata.Namespace = spec.Ref.Namespace
			m.Metadata.Name = spec.Ref.Name
			if err := writeManifestYAML(manifestPath, m); err != nil {
				return err
			}
		}
	}
	fmt.Printf("Added remote %s -> %s/%s\n", name, spec.ServerURL, spec.Package)
	return nil
}

func remoteList(jsonOutput bool) error {
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", projectConfigPath, err)
	}
	names := sortedRemoteNames(cfg.Remotes)
	if jsonOutput {
		items := make([]map[string]string, 0, len(names))
		for _, name := range names {
			remote := cfg.Remotes[name]
			items = append(items, map[string]string{"name": name, "serverUrl": remote.ServerURL, "package": remote.Package})
		}
		return outputJSON(map[string]any{"items": items})
	}
	for _, name := range names {
		remote := cfg.Remotes[name]
		marker := ""
		if name == cfg.DefaultRemote {
			marker = " (default)"
		}
		fmt.Printf("%s -> %s/%s%s\n", name, remote.ServerURL, remote.Package, marker)
	}
	return nil
}

func remoteRemove(name string) error {
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", projectConfigPath, err)
	}
	if _, ok := cfg.Remotes[name]; !ok {
		return fmt.Errorf("remote %q does not exist", name)
	}
	delete(cfg.Remotes, name)
	if cfg.DefaultRemote == name {
		cfg.DefaultRemote = ""
	}
	if len(cfg.Remotes) == 0 {
		cfg.Remotes = nil
	}
	return writeProjectConfig(projectConfigPath, cfg)
}
