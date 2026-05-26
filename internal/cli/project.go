package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"trove/internal/manifest"
)

const (
	projectConfigPath = ".trove/config.yaml"
	projectStatePath  = ".trove/state.yaml"
	projectAPIVersion = "trove.io/v1"
	projectKind       = "TroveProject"
	projectStateKind  = "TroveProjectState"
	agentsMDKind      = "agents-md"
	agentsMDPath      = "AGENTS.md"
	manifestPath      = "trove.yaml"
)

type ProjectConfig struct {
	APIVersion    string                   `yaml:"apiVersion" json:"apiVersion"`
	Kind          string                   `yaml:"kind" json:"kind"`
	DefaultRemote string                   `yaml:"defaultRemote,omitempty" json:"defaultRemote,omitempty"`
	ArtifactKind  string                   `yaml:"artifactKind" json:"artifactKind"`
	Remotes       map[string]ProjectRemote `yaml:"remotes,omitempty" json:"remotes,omitempty"`
	Publish       PublishConfig            `yaml:"publish" json:"publish"`
}

type ProjectRemote struct {
	ServerURL string `yaml:"serverUrl" json:"serverUrl"`
	Package   string `yaml:"package" json:"package"`
}

type PublishConfig struct {
	Visibility string `yaml:"visibility" json:"visibility"`
}

type ProjectState struct {
	APIVersion string               `yaml:"apiVersion" json:"apiVersion"`
	Kind       string               `yaml:"kind" json:"kind"`
	Source     ProjectStateSource   `yaml:"source" json:"source"`
	Files      map[string]StateFile `yaml:"files" json:"files"`
}

type ProjectStateSource struct {
	Remote            string `yaml:"remote" json:"remote"`
	RequestedSelector string `yaml:"requestedSelector" json:"requestedSelector"`
	ResolvedVersion   string `yaml:"resolvedVersion" json:"resolvedVersion"`
	PackageDigest     string `yaml:"packageDigest" json:"packageDigest"`
	ManifestDigest    string `yaml:"manifestDigest,omitempty" json:"manifestDigest,omitempty"`
}

type StateFile struct {
	Digest string `yaml:"digest" json:"digest"`
}

type RemoteSpec struct {
	ServerURL string
	Package   string
	Ref       PackageRef
}

type SemVer struct {
	Major int
	Minor int
	Patch int
}

var strictSemverRE = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)$`)

func loadProjectConfig(path string) (ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, err
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func writeProjectConfig(path string, cfg ProjectConfig) error {
	if cfg.APIVersion == "" {
		cfg.APIVersion = projectAPIVersion
	}
	if cfg.Kind == "" {
		cfg.Kind = projectKind
	}
	if cfg.ArtifactKind == "" {
		cfg.ArtifactKind = agentsMDKind
	}
	if cfg.Publish.Visibility == "" {
		cfg.Publish.Visibility = "private"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func validateProjectConfig(cfg ProjectConfig) []string {
	var problems []string
	if cfg.APIVersion != projectAPIVersion {
		problems = append(problems, "apiVersion must be trove.io/v1")
	}
	if cfg.Kind != projectKind {
		problems = append(problems, "kind must be TroveProject")
	}
	if cfg.ArtifactKind != agentsMDKind {
		problems = append(problems, "artifactKind must be agents-md")
	}
	if cfg.DefaultRemote != "" {
		if cfg.Remotes == nil {
			problems = append(problems, "defaultRemote is set but remotes is empty")
		} else if _, ok := cfg.Remotes[cfg.DefaultRemote]; !ok {
			problems = append(problems, "defaultRemote must exist in remotes")
		}
	}
	for name, remote := range cfg.Remotes {
		if strings.TrimSpace(name) == "" {
			problems = append(problems, "remote name is required")
		}
		if _, err := ParsePackageRefNoSelector(remote.Package); err != nil {
			problems = append(problems, fmt.Sprintf("remote %s package is invalid: %v", name, err))
		}
		if err := validateServerURL(remote.ServerURL); err != nil {
			problems = append(problems, fmt.Sprintf("remote %s serverUrl is invalid: %v", name, err))
		}
	}
	return problems
}

func loadProjectState(path string) (ProjectState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectState{}, err
	}
	var state ProjectState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return ProjectState{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return state, nil
}

func writeProjectState(path string, state ProjectState) error {
	if state.APIVersion == "" {
		state.APIVersion = projectAPIVersion
	}
	if state.Kind == "" {
		state.Kind = projectStateKind
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func validateProjectState(state ProjectState) []string {
	var problems []string
	if state.APIVersion != projectAPIVersion {
		problems = append(problems, "apiVersion must be trove.io/v1")
	}
	if state.Kind != projectStateKind {
		problems = append(problems, "kind must be TroveProjectState")
	}
	if state.Source.Remote == "" {
		problems = append(problems, "source.remote is required")
	}
	if state.Source.ResolvedVersion == "" {
		problems = append(problems, "source.resolvedVersion is required")
	}
	if len(state.Files) == 0 {
		problems = append(problems, "files must include baseline digests")
	}
	for path, file := range state.Files {
		if path == "" {
			problems = append(problems, "file path is required")
		}
		if file.Digest == "" {
			problems = append(problems, fmt.Sprintf("files.%s.digest is required", path))
		}
	}
	return problems
}

func parseRemoteSpec(raw string, existing ProjectConfig) (RemoteSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RemoteSpec{}, fmt.Errorf("remote value is required")
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return parseFullRemoteURL(raw)
	}

	ref, err := ParsePackageRefNoSelector(raw)
	if err != nil {
		return RemoteSpec{}, err
	}
	serverURL := os.Getenv("TROVE_SERVER_URL")
	if serverURL == "" && existing.DefaultRemote != "" {
		if remote, ok := existing.Remotes[existing.DefaultRemote]; ok {
			serverURL = remote.ServerURL
		}
	}
	if serverURL == "" {
		return RemoteSpec{}, fmt.Errorf("short package refs require TROVE_SERVER_URL or an existing configured remote")
	}
	if err := validateServerURL(serverURL); err != nil {
		return RemoteSpec{}, err
	}
	return RemoteSpec{ServerURL: strings.TrimRight(serverURL, "/"), Package: ref.PackagePath(), Ref: ref}, nil
}

func parseFullRemoteURL(raw string) (RemoteSpec, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return RemoteSpec{}, fmt.Errorf("invalid remote URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return RemoteSpec{}, fmt.Errorf("remote URL scheme must be http or https")
	}
	if u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return RemoteSpec{}, fmt.Errorf("remote URL must be an absolute package URL without query or fragment")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 {
		return RemoteSpec{}, fmt.Errorf("remote URL path must be /org/namespace/package")
	}
	ref, err := ParsePackageRefNoSelector(strings.Join(parts, "/"))
	if err != nil {
		return RemoteSpec{}, err
	}
	u.Path = ""
	return RemoteSpec{ServerURL: strings.TrimRight(u.String(), "/"), Package: ref.PackagePath(), Ref: ref}, nil
}

func validateServerURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("must be absolute")
	}
	return nil
}

func remoteForConfig(cfg ProjectConfig, selected string) (string, ProjectRemote, error) {
	name := selected
	if name == "" {
		name = cfg.DefaultRemote
	}
	if name == "" {
		return "", ProjectRemote{}, fmt.Errorf("missing remote; run 'trove remote add origin <url-or-package-ref>' or rerun init with --remote")
	}
	remote, ok := cfg.Remotes[name]
	if !ok {
		return "", ProjectRemote{}, fmt.Errorf("remote %q is not configured", name)
	}
	return name, remote, nil
}

func ParsePackageRefNoSelector(raw string) (PackageRef, error) {
	ref, err := ParsePackageRef(raw)
	if err != nil {
		return PackageRef{}, err
	}
	if strings.Contains(raw, "@") {
		return PackageRef{}, fmt.Errorf("remote package reference must not include a selector")
	}
	ref.Selector = ""
	return ref, nil
}

func ParseSemVer(raw string) (SemVer, error) {
	m := strictSemverRE.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return SemVer{}, fmt.Errorf("version must be strict SemVer MAJOR.MINOR.PATCH")
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func nextVersion(versions []PackageVersion, bump string, explicit string) (string, error) {
	if explicit != "" {
		v, err := ParseSemVer(explicit)
		if err != nil {
			return "", err
		}
		return v.String(), nil
	}
	var max *SemVer
	for _, version := range versions {
		parsed, err := ParseSemVer(version.Version)
		if err != nil {
			continue
		}
		if max == nil || semverLess(*max, parsed) {
			copy := parsed
			max = &copy
		}
	}
	if max == nil {
		return "1.0.0", nil
	}
	switch bump {
	case "", "patch":
		max.Patch++
	case "minor":
		max.Minor++
		max.Patch = 0
	case "major":
		max.Major++
		max.Minor = 0
		max.Patch = 0
	default:
		return "", fmt.Errorf("unknown version bump %q", bump)
	}
	return max.String(), nil
}

func semverLess(a, b SemVer) bool {
	if a.Major != b.Major {
		return a.Major < b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor < b.Minor
	}
	return a.Patch < b.Patch
}

func packageVersionsForRemote(remote ProjectRemote) ([]PackageVersion, *PackageDetailResponse, error) {
	ref, err := ParsePackageRefNoSelector(remote.Package)
	if err != nil {
		return nil, nil, err
	}
	client := NewClientForServer(remote.ServerURL)
	detail, err := client.GetPackage(ref.Org, ref.Namespace, ref.Name)
	if err != nil {
		return nil, nil, err
	}
	return detail.Versions, detail, nil
}

func generatedAgentsManifest(ref PackageRef, displayName, description, visibility string, maintainers []manifest.Maintainer) manifest.Manifest {
	if displayName == "" {
		displayName = titleFromSlug(ref.Name)
	}
	if description == "" {
		description = "Shared AGENTS.md instructions."
	}
	if visibility == "" {
		visibility = "private"
	}
	return manifest.Manifest{
		APIVersion: manifest.APIVersion,
		Kind:       manifest.Kind,
		Metadata: manifest.Metadata{
			Org:         ref.Org,
			Namespace:   ref.Namespace,
			Name:        ref.Name,
			DisplayName: displayName,
			Description: description,
		},
		Spec: manifest.Spec{
			Version:     "1.0.0",
			Visibility:  visibility,
			Lifecycle:   "draft",
			Artifacts:   agentsArtifacts(),
			Maintainers: maintainers,
		},
	}
}

func agentsArtifacts() []manifest.Artifact {
	return []manifest.Artifact{{Path: agentsMDPath, Type: "agent-instructions", Required: true, TargetPath: agentsMDPath}}
}

func validateAgentsManifest(m manifest.Manifest) []string {
	var problems []string
	if err := manifest.Validate(m, manifest.ValidateOptions{}); err != nil {
		var manifestErr *manifest.Error
		if errors.As(err, &manifestErr) {
			for _, problem := range manifestErr.Problems {
				problems = append(problems, problem.Field+" "+problem.Message)
			}
		} else {
			problems = append(problems, err.Error())
		}
	}
	found := false
	for _, artifact := range m.Spec.Artifacts {
		if artifact.Path != agentsMDPath {
			continue
		}
		found = true
		if artifact.Type != "agent-instructions" {
			problems = append(problems, "spec.artifacts AGENTS.md type must be agent-instructions")
		}
		if !artifact.Required {
			problems = append(problems, "spec.artifacts AGENTS.md required must be true")
		}
		if artifact.TargetPath != "" && artifact.TargetPath != agentsMDPath {
			problems = append(problems, "spec.artifacts AGENTS.md targetPath must be AGENTS.md")
		}
	}
	if !found {
		problems = append(problems, "spec.artifacts must include AGENTS.md")
	}
	return problems
}

func writeManifestYAML(path string, m manifest.Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func loadManifestYAML(path string) (manifest.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest.Manifest{}, err
	}
	return manifest.Parse(data)
}

func manifestYAMLBytes(m manifest.Manifest) ([]byte, error) {
	return yaml.Marshal(m)
}

func titleFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func slugFromDir(dir string) string {
	base := strings.ToLower(filepath.Base(dir))
	var b strings.Builder
	lastDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) < 3 {
		return "agent-defaults"
	}
	if len(slug) > 63 {
		slug = strings.Trim(slug[:63], "-")
	}
	return slug
}

func configWithRemote(spec RemoteSpec) ProjectConfig {
	return ProjectConfig{
		APIVersion:    projectAPIVersion,
		Kind:          projectKind,
		DefaultRemote: "origin",
		ArtifactKind:  agentsMDKind,
		Remotes: map[string]ProjectRemote{
			"origin": {ServerURL: spec.ServerURL, Package: spec.Package},
		},
		Publish: PublishConfig{Visibility: "private"},
	}
}

func sortedRemoteNames(remotes map[string]ProjectRemote) []string {
	names := make([]string, 0, len(remotes))
	for name := range remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return computeDigest(data), nil
}

func stateMatches(path string, expected StateFile) bool {
	if expected.Digest == "" {
		return false
	}
	digest, err := fileDigest(path)
	return err == nil && digest == expected.Digest
}

func writeIfClean(path string, content []byte, state ProjectState) error {
	if err := ensureCleanForWrite(path, state); err != nil {
		return err
	}
	return writeFileContent(path, content)
}

func ensureCleanForWrite(path string, state ProjectState) error {
	baseline, ok := state.Files[path]
	if !ok || baseline.Digest == "" {
		return fmt.Errorf("missing baseline for %s; refusing to overwrite", path)
	}
	if _, err := os.Stat(path); err == nil && !stateMatches(path, baseline) {
		return fmt.Errorf("local changes conflict for %s; refusing to overwrite", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeFileContent(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func marshalRawManifestAsYAML(raw []byte) ([]byte, manifest.Manifest, error) {
	var m manifest.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		if parsed, parseErr := manifest.Parse(raw); parseErr == nil {
			m = parsed
		} else {
			return nil, manifest.Manifest{}, fmt.Errorf("parse manifest: %w", err)
		}
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, manifest.Manifest{}, err
	}
	return data, m, nil
}

func isAgentsMDManifest(m manifest.Manifest) bool {
	return len(validateAgentsManifest(m)) == 0 && len(m.Spec.Artifacts) == 1 && m.Spec.Artifacts[0].Path == agentsMDPath
}

func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readOptionalProjectConfig() ProjectConfig {
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		return ProjectConfig{}
	}
	return cfg
}

func readManifestIfExists() (manifest.Manifest, bool) {
	m, err := loadManifestYAML(manifestPath)
	return m, err == nil
}

func shouldWriteGenerated(path string, force bool) bool {
	if force {
		return true
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func sameBytesFile(path string, data []byte) bool {
	existing, err := os.ReadFile(path)
	return err == nil && bytes.Equal(existing, data)
}
