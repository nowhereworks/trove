package cli

import (
	"fmt"

	"trove/internal/manifest"
)

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
	m := readOptionalTrovefile()
	if m.APIVersion == "" {
		m = manifest.Manifest{APIVersion: manifest.APIVersion, Kind: manifest.Kind}
	}
	spec, err := parseRemoteSpec(value, m)
	if err != nil {
		return err
	}
	if m.Local == nil {
		m.Local = &manifest.Local{}
	}
	if m.Local.Remotes == nil {
		m.Local.Remotes = map[string]manifest.ProjectRemote{}
	}
	if _, exists := m.Local.Remotes[name]; exists && !force {
		return fmt.Errorf("remote %q already exists; use --force to replace", name)
	}
	m.Local.Remotes[name] = manifest.ProjectRemote{ServerURL: spec.ServerURL, Package: spec.Package}
	if m.Local.DefaultRemote == "" {
		m.Local.DefaultRemote = name
	}
	if err := writeTrovefile(manifestPath, m); err != nil {
		return err
	}
	if m.Metadata.Org == "" || m.Metadata.Org == "nwks" {
		m.Metadata.Org = spec.Ref.Org
		m.Metadata.Namespace = spec.Ref.Namespace
		m.Metadata.Name = spec.Ref.Name
		if err := writeTrovefile(manifestPath, m); err != nil {
			return err
		}
	}
	fmt.Printf("Added remote %s -> %s/%s\n", name, spec.ServerURL, spec.Package)
	return nil
}

func remoteList(jsonOutput bool) error {
	m, err := loadTrovefile(manifestPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", manifestPath, err)
	}
	if m.Local == nil || m.Local.Remotes == nil {
		if jsonOutput {
			return outputJSON(map[string]any{"items": []map[string]string{}})
		}
		return nil
	}
	names := sortedRemoteNames(m.Local.Remotes)
	if jsonOutput {
		items := make([]map[string]string, 0, len(names))
		for _, name := range names {
			remote := m.Local.Remotes[name]
			items = append(items, map[string]string{"name": name, "serverUrl": remote.ServerURL, "package": remote.Package})
		}
		return outputJSON(map[string]any{"items": items})
	}
	for _, name := range names {
		remote := m.Local.Remotes[name]
		marker := ""
		if name == m.Local.DefaultRemote {
			marker = " (default)"
		}
		fmt.Printf("%s -> %s/%s%s\n", name, remote.ServerURL, remote.Package, marker)
	}
	return nil
}

func remoteRemove(name string) error {
	m, err := loadTrovefile(manifestPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", manifestPath, err)
	}
	if m.Local == nil || m.Local.Remotes == nil {
		return fmt.Errorf("remote %q does not exist", name)
	}
	if _, ok := m.Local.Remotes[name]; !ok {
		return fmt.Errorf("remote %q does not exist", name)
	}
	delete(m.Local.Remotes, name)
	if m.Local.DefaultRemote == name {
		m.Local.DefaultRemote = ""
	}
	if len(m.Local.Remotes) == 0 {
		m.Local.Remotes = nil
	}
	return writeTrovefile(manifestPath, m)
}
