package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func RunClone(args []string, jsonOutput bool) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove clone <org/namespace/package[@selector]> [dir]")
	}
	ref, err := ParsePackageRef(args[0])
	if err != nil {
		return err
	}
	dir := ref.Name
	if len(args) > 1 {
		dir = args[1]
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("target directory %s exists and is not empty", dir)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	client := NewClient()
	resolved, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, ref.Selector)
	if err != nil {
		return err
	}
	manifestResp, err := client.GetManifest(ref.Org, ref.Namespace, ref.Name, resolved.ResolvedVersion)
	if err != nil {
		return err
	}
	manifestBytes, m, err := marshalRawManifestAsYAML(manifestResp.Raw)
	if err != nil {
		return err
	}
	manifestOut := filepath.Join(dir, manifestPath)
	if err := os.WriteFile(manifestOut, manifestBytes, 0644); err != nil {
		return err
	}

	manifestDigest := computeDigest(manifestBytes)
	state := ProjectState{APIVersion: projectAPIVersion, Kind: projectStateKind, Source: ProjectStateSource{Remote: "origin", RequestedSelector: ref.Selector, ResolvedVersion: resolved.ResolvedVersion, PackageDigest: resolved.Digest, ManifestDigest: manifestDigest}, Files: map[string]StateFile{manifestPath: {Digest: manifestDigest}}}
	for _, artifact := range m.Spec.Artifacts {
		content, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, resolved.ResolvedVersion, artifact.Path)
		if err != nil {
			return fmt.Errorf("download %s: %w", artifact.Path, err)
		}
		path := filepath.Join(dir, artifact.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			return err
		}
		state.Files[artifact.Path] = StateFile{Digest: computeDigest(content)}
	}

	cfg := configWithRemote(RemoteSpec{ServerURL: client.ServerURL, Package: ref.PackagePath(), Ref: PackageRef{Org: ref.Org, Namespace: ref.Namespace, Name: ref.Name}})
	if !isAgentsMDManifest(m) {
		cfg.ArtifactKind = ""
	}
	if err := writeProjectConfig(filepath.Join(dir, projectConfigPath), cfg); err != nil {
		return err
	}
	if err := writeProjectState(filepath.Join(dir, projectStatePath), state); err != nil {
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]string{"package": ref.PackagePath(), "selector": ref.Selector, "version": resolved.ResolvedVersion, "output": dir})
	}
	fmt.Printf("Cloned %s@%s to %s\n", ref.PackagePath(), resolved.ResolvedVersion, dir)
	return nil
}

func RunPull(args []string, jsonOutput bool) error {
	remoteNameFlag := flagValue(args, "--remote")
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", projectConfigPath, err)
	}
	remoteName, remote, err := remoteForConfig(cfg, remoteNameFlag)
	if err != nil {
		return err
	}
	state, err := loadProjectState(projectStatePath)
	if err != nil {
		return fmt.Errorf("missing or incomplete %s; refusing to overwrite existing files", projectStatePath)
	}
	if problems := validateProjectState(state); len(problems) > 0 {
		return fmt.Errorf("missing or incomplete %s: %s; refusing to overwrite existing files", projectStatePath, problems[0])
	}
	if state.Source.Remote != "" {
		remoteName = state.Source.Remote
		remote = cfg.Remotes[remoteName]
		if remote.ServerURL == "" || remote.Package == "" {
			return fmt.Errorf("remote %q from %s is not configured", remoteName, projectStatePath)
		}
	}
	selector := state.Source.RequestedSelector
	if selector == "" {
		selector = "stable"
	}
	ref, err := ParsePackageRefNoSelector(remote.Package)
	if err != nil {
		return err
	}
	client := NewClientForServer(remote.ServerURL)
	resolved, err := client.Resolve(ref.Org, ref.Namespace, ref.Name, selector)
	if err != nil {
		return err
	}
	manifestResp, err := client.GetManifest(ref.Org, ref.Namespace, ref.Name, resolved.ResolvedVersion)
	if err != nil {
		return err
	}
	manifestBytes, m, err := marshalRawManifestAsYAML(manifestResp.Raw)
	if err != nil {
		return err
	}
	contents := map[string][]byte{manifestPath: manifestBytes}
	for _, artifact := range m.Spec.Artifacts {
		content, err := client.GetRawArtifact(ref.Org, ref.Namespace, ref.Name, resolved.ResolvedVersion, artifact.Path)
		if err != nil {
			return err
		}
		contents[artifact.Path] = content
	}
	for path, content := range contents {
		if sameBytesFile(path, content) {
			continue
		}
		if err := ensureCleanForWrite(path, state); err != nil {
			return err
		}
	}
	for path, content := range contents {
		if sameBytesFile(path, content) {
			continue
		}
		if err := writeFileContent(path, content); err != nil {
			return err
		}
	}

	newState := ProjectState{APIVersion: projectAPIVersion, Kind: projectStateKind, Source: ProjectStateSource{Remote: remoteName, RequestedSelector: selector, ResolvedVersion: resolved.ResolvedVersion, PackageDigest: resolved.Digest, ManifestDigest: computeDigest(manifestBytes)}, Files: map[string]StateFile{}}
	for path, content := range contents {
		newState.Files[path] = StateFile{Digest: computeDigest(content)}
	}
	if err := writeProjectState(projectStatePath, newState); err != nil {
		return err
	}
	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"package": ref.PackagePath(), "version": resolved.ResolvedVersion})
	}
	fmt.Printf("Pulled %s@%s\n", ref.PackagePath(), resolved.ResolvedVersion)
	return nil
}
