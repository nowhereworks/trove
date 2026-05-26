package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitRequiresForceWhenManifestExists(t *testing.T) {
	chdirTemp(t)
	t.Setenv("TROVE_SERVER_URL", "")
	if err := os.WriteFile(manifestPath, []byte("apiVersion: trove.io/v1\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	err := RunInit([]string{"agents-md", "--package", "nwks/platform/test"}, false)
	if err == nil {
		t.Fatalf("expected error when trove.yaml exists without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want 'already exists'", err)
	}
}

func TestInitPackageOnlyCreatesManifestWithoutRemote(t *testing.T) {
	chdirTemp(t)
	t.Setenv("TROVE_SERVER_URL", "")

	if err := RunInit([]string{"agents-md", "--package", "nwks/platform/agent-defaults", "--yes"}, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	agents, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agents), "Shared agent instructions") {
		t.Fatalf("starter AGENTS.md = %q", agents)
	}
	m, err := loadManifestYAML(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if m.Metadata.Org != "nwks" || m.Metadata.Namespace != "platform" || m.Metadata.Name != "agent-defaults" {
		t.Fatalf("manifest package = %s/%s/%s", m.Metadata.Org, m.Metadata.Namespace, m.Metadata.Name)
	}
	if len(m.Spec.Artifacts) != 1 || m.Spec.Artifacts[0].Path != agentsMDPath {
		t.Fatalf("manifest artifacts = %+v", m.Spec.Artifacts)
	}
	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DefaultRemote != "" || len(cfg.Remotes) != 0 {
		t.Fatalf("config remote = %q/%+v, want none", cfg.DefaultRemote, cfg.Remotes)
	}
}

func TestInitPackageOnlyUsesExistingConfigServer(t *testing.T) {
	chdirTemp(t)
	t.Setenv("TROVE_SERVER_URL", "")
	if err := writeProjectConfig(projectConfigPath, configWithRemote(RemoteSpec{ServerURL: "https://trove.company.com", Package: "nwks/platform/old-defaults", Ref: PackageRef{Org: "nwks", Namespace: "platform", Name: "old-defaults"}})); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	if err := RunInit([]string{"agents-md", "--package", "nwks/platform/agent-defaults", "--force", "--yes"}, false); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	remote := cfg.Remotes["origin"]
	if cfg.DefaultRemote != "origin" || remote.ServerURL != "https://trove.company.com" || remote.Package != "nwks/platform/agent-defaults" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestRemoteAddShortRefUsesEnvServer(t *testing.T) {
	chdirTemp(t)
	t.Setenv("TROVE_SERVER_URL", "https://trove.company.com/")

	if err := RunRemote([]string{"add", "origin", "nwks/platform/agent-defaults"}, false, false); err != nil {
		t.Fatalf("RunRemote(add) error = %v", err)
	}

	cfg, err := loadProjectConfig(projectConfigPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	remote := cfg.Remotes["origin"]
	if cfg.DefaultRemote != "origin" || remote.ServerURL != "https://trove.company.com" || remote.Package != "nwks/platform/agent-defaults" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestStatusReadyComputesNextPatchVersion(t *testing.T) {
	chdirTemp(t)
	server := newPackageDetailServer(t, []PackageVersion{{Version: "1.0.0", Lifecycle: "published"}})
	t.Setenv("TROVE_TOKEN", "test-token")

	if err := writeProjectConfig(projectConfigPath, configWithRemote(RemoteSpec{ServerURL: server.URL, Package: "nwks/platform/agent-defaults", Ref: PackageRef{Org: "nwks", Namespace: "platform", Name: "agent-defaults"}})); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(manifestPath, agentsManifestBytes(t, "1.0.0"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(agentsMDPath, []byte("# Local instructions\n"), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	stdout := captureStdout(t, func() error {
		return RunStatus(nil, true)
	})
	var out statusOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode status JSON %q: %v", stdout, err)
	}
	if out.LocalState != "ready" || out.CurrentPublishedVersion != "1.0.0" || out.NextVersion != "1.0.1" || len(out.Problems) != 0 {
		t.Fatalf("status output = %+v", out)
	}
}

func TestDownloadWritesOutputAndNoProjectMetadata(t *testing.T) {
	chdirTemp(t)
	server := newAgentsMDServer(t, "1.0.0", "# Downloaded defaults\n")
	t.Setenv("TROVE_SERVER_URL", server.URL)

	stdout := captureStdout(t, func() error {
		return RunDownload([]string{"nwks/platform/agent-defaults", agentsMDPath}, filepath.Join("out", agentsMDPath), false, false, true)
	})
	var out map[string]string
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode download JSON %q: %v", stdout, err)
	}
	if out["version"] != "1.0.0" || out["artifactPath"] != agentsMDPath || out["changed"] != "true" {
		t.Fatalf("download output = %+v", out)
	}
	content, err := os.ReadFile(filepath.Join("out", agentsMDPath))
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) != "# Downloaded defaults\n" {
		t.Fatalf("downloaded content = %q", content)
	}
	for _, unexpected := range []string{manifestPath, projectConfigPath, ".trove.lock.yaml"} {
		if _, err := os.Stat(unexpected); !os.IsNotExist(err) {
			t.Fatalf("%s exists after download; err=%v", unexpected, err)
		}
	}
}

func TestDownloadCoreSkillWritesOutput(t *testing.T) {
	chdirTemp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/core/skills/find-trove-skills/SKILL.md" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte("# find-trove-skills\n"))
	}))
	t.Cleanup(server.Close)
	t.Setenv("TROVE_SERVER_URL", server.URL)

	stdout := captureStdout(t, func() error {
		return RunDownload([]string{"core/skills/find-trove-skills/SKILL.md"}, filepath.Join("skills", "find-trove-skills", "SKILL.md"), false, false, true)
	})
	var out map[string]string
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode download JSON %q: %v", stdout, err)
	}
	if out["source"] != "core" || out["coreSkill"] != "find-trove-skills" || out["artifactPath"] != "core/skills/find-trove-skills/SKILL.md" || out["changed"] != "true" {
		t.Fatalf("download output = %+v", out)
	}
	content, err := os.ReadFile(filepath.Join("skills", "find-trove-skills", "SKILL.md"))
	if err != nil {
		t.Fatalf("read downloaded skill: %v", err)
	}
	if string(content) != "# find-trove-skills\n" {
		t.Fatalf("downloaded skill = %q", content)
	}
}

func TestParseCoreSkillDownloadPath(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "core/skills/find-trove-skills/SKILL.md", want: "find-trove-skills", ok: true},
		{path: "core/skills//SKILL.md", ok: false},
		{path: "core/skills/find-trove-skills/extra/SKILL.md", ok: false},
		{path: "core/skills/find-trove-skills/README.md", ok: false},
		{path: "core/skills/../SKILL.md", ok: false},
		{path: "core/skills/find@trove/SKILL.md", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, ok := parseCoreSkillDownloadPath(tt.path)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseCoreSkillDownloadPath(%q) = %q, %v; want %q, %v", tt.path, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func newPackageDetailServer(t *testing.T, versions []PackageVersion) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.1/approval-status" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ApprovalStatusResponse{HasEnoughApprovals: false, CurrentCount: 0, RequiredCount: 1})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/packages/nwks/platform/agent-defaults" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PackageDetailResponse{Org: "nwks", Namespace: "platform", Name: "agent-defaults", Visibility: "private", Versions: versions})
	}))
	t.Cleanup(server.Close)
	return server
}
