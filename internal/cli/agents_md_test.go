package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trove/internal/manifest"
)

func TestCloneCreatesAgentsMDWorktree(t *testing.T) {
	chdirTemp(t)
	server := newAgentsMDServer(t, "1.0.0", "# Agent defaults\n")
	t.Setenv("TROVE_SERVER_URL", server.URL)

	if err := RunClone([]string{"nwks/platform/agent-defaults@latest"}, false); err != nil {
		t.Fatalf("RunClone() error = %v", err)
	}

	agents, err := os.ReadFile(filepath.Join("agent-defaults", agentsMDPath))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(agents) != "# Agent defaults\n" {
		t.Fatalf("AGENTS.md = %q", agents)
	}
	if _, err := os.Stat(filepath.Join("agent-defaults", manifestPath)); err != nil {
		t.Fatalf("missing manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join("agent-defaults", projectConfigPath)); err != nil {
		t.Fatalf("missing project config: %v", err)
	}
	state, err := loadProjectState(filepath.Join("agent-defaults", projectStatePath))
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Source.Remote != "origin" || state.Source.ResolvedVersion != "1.0.0" {
		t.Fatalf("state source = %+v", state.Source)
	}
	if state.Source.ManifestDigest == "" || state.Files[agentsMDPath].Digest == "" || state.Files[manifestPath].Digest == "" {
		t.Fatalf("state file digests missing: %+v", state.Files)
	}
}

func TestPullUsesStateSourceRemote(t *testing.T) {
	chdirTemp(t)
	origin := newAgentsMDServer(t, "1.0.1", "# Updated defaults\n")
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("pull contacted default remote instead of state source remote: %s", r.URL.Path)
	}))
	t.Cleanup(backup.Close)

	oldManifest := agentsManifestBytes(t, "1.0.0")
	oldAgents := []byte("# Old defaults\n")
	if err := os.WriteFile(manifestPath, oldManifest, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(agentsMDPath, oldAgents, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := writeProjectConfig(projectConfigPath, ProjectConfig{
		APIVersion:    projectAPIVersion,
		Kind:          projectKind,
		DefaultRemote: "backup",
		ArtifactKind:  agentsMDKind,
		Remotes: map[string]ProjectRemote{
			"origin": {ServerURL: origin.URL, Package: "nwks/platform/agent-defaults"},
			"backup": {ServerURL: backup.URL, Package: "nwks/platform/agent-defaults"},
		},
	}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := writeProjectState(projectStatePath, ProjectState{
		APIVersion: projectAPIVersion,
		Kind:       projectStateKind,
		Source: ProjectStateSource{
			Remote:            "origin",
			RequestedSelector: "latest",
			ResolvedVersion:   "1.0.0",
			PackageDigest:     "sha256:old",
		},
		Files: map[string]StateFile{
			manifestPath: {Digest: computeDigest(oldManifest)},
			agentsMDPath: {Digest: computeDigest(oldAgents)},
		},
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	if err := RunPull(nil, false); err != nil {
		t.Fatalf("RunPull() error = %v", err)
	}

	updated, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(updated) != "# Updated defaults\n" {
		t.Fatalf("AGENTS.md = %q", updated)
	}
	state, err := loadProjectState(projectStatePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Source.Remote != "origin" || state.Source.ResolvedVersion != "1.0.1" || state.Source.ManifestDigest == "" {
		t.Fatalf("state source = %+v", state.Source)
	}
}

func TestPullRefusesIncompleteState(t *testing.T) {
	chdirTemp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("pull contacted remote with incomplete state: %s", r.URL.Path)
	}))
	t.Cleanup(server.Close)

	if err := writeProjectConfig(projectConfigPath, configWithRemote(RemoteSpec{ServerURL: server.URL, Package: "nwks/platform/agent-defaults", Ref: PackageRef{Org: "nwks", Namespace: "platform", Name: "agent-defaults"}})); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := writeProjectState(projectStatePath, ProjectState{APIVersion: projectAPIVersion, Kind: projectStateKind}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	err := RunPull(nil, false)
	if err == nil || !strings.Contains(err.Error(), "missing or incomplete .trove/state.yaml") {
		t.Fatalf("RunPull() error = %v, want incomplete state error", err)
	}
}

func TestPullReportsConflictsWithoutPartialWrites(t *testing.T) {
	chdirTemp(t)
	server := newAgentsMDServer(t, "1.0.1", "# Remote defaults\n")

	oldManifest := agentsManifestBytes(t, "1.0.0")
	oldAgents := []byte("# Old defaults\n")
	localAgents := []byte("# Local edits\n")
	if err := os.WriteFile(manifestPath, oldManifest, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(agentsMDPath, localAgents, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := writeProjectConfig(projectConfigPath, configWithRemote(RemoteSpec{ServerURL: server.URL, Package: "nwks/platform/agent-defaults", Ref: PackageRef{Org: "nwks", Namespace: "platform", Name: "agent-defaults"}})); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := writeProjectState(projectStatePath, ProjectState{
		APIVersion: projectAPIVersion,
		Kind:       projectStateKind,
		Source: ProjectStateSource{
			Remote:            "origin",
			RequestedSelector: "latest",
			ResolvedVersion:   "1.0.0",
			PackageDigest:     "sha256:old",
		},
		Files: map[string]StateFile{
			manifestPath: {Digest: computeDigest(oldManifest)},
			agentsMDPath: {Digest: computeDigest(oldAgents)},
		},
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	err := RunPull(nil, false)
	if err == nil || !strings.Contains(err.Error(), "local changes conflict for AGENTS.md") {
		t.Fatalf("RunPull() error = %v, want AGENTS.md conflict", err)
	}
	manifestAfter, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if string(manifestAfter) != string(oldManifest) {
		t.Fatalf("manifest was partially updated before conflict")
	}
	agentsAfter, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(agentsAfter) != string(localAgents) {
		t.Fatalf("AGENTS.md was overwritten despite conflict: %q", agentsAfter)
	}
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
	return dir
}

func newAgentsMDServer(t *testing.T, version string, agentsContent string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/resolve/nwks/platform/agent-defaults@latest":
			_ = json.NewEncoder(w).Encode(ResolveResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Selector: "latest", ResolvedVersion: version, Digest: "sha256:" + version})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/"+version+"/manifest":
			_ = json.NewEncoder(w).Encode(ManifestResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: version, Digest: "sha256:" + version, Manifest: agentsManifestJSON(t, version)})
		case r.Method == http.MethodGet && r.URL.Path == "/raw/nwks/platform/agent-defaults/AGENTS.md@"+version:
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte(agentsContent))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func agentsManifestJSON(t *testing.T, version string) json.RawMessage {
	t.Helper()
	m := agentsManifest(version)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest JSON: %v", err)
	}
	return data
}

func agentsManifestBytes(t *testing.T, version string) []byte {
	t.Helper()
	data, err := manifestYAMLBytes(agentsManifest(version))
	if err != nil {
		t.Fatalf("marshal manifest YAML: %v", err)
	}
	return data
}

func agentsManifest(version string) manifest.Manifest {
	m := generatedAgentsManifest(PackageRef{Org: "nwks", Namespace: "platform", Name: "agent-defaults"}, "Agent Defaults", "Shared AGENTS.md instructions.")
	m.Spec.Lifecycle = "published"
	return m
}
