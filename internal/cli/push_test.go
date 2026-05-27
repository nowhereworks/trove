package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"trove/internal/manifest"
)

func TestPushNewPackagePublishesAndUploadsManifestFirst(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	if err := RunPush(nil, false); err != nil {
		t.Fatalf("RunPush() error = %v", err)
	}

	want := []string{"GET package", "POST draft 1.0.0", "PUT Trovefile", "PUT AGENTS.md", "POST publish"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %#v, want %#v", *calls, want)
	}
	agents, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(agents) != "# Local instructions\n" {
		t.Fatalf("AGENTS.md was rewritten: %q", agents)
	}
	m, err := loadTrovefile(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	_ = m
}

func TestPushExistingPackageUsesNextPatchVersion(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{versions: []PackageVersion{{Version: "1.0.0", Lifecycle: "published"}}})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	if err := RunPush(nil, false); err != nil {
		t.Fatalf("RunPush() error = %v", err)
	}

	want := []string{"GET package", "POST draft 1.0.1", "PUT Trovefile", "PUT AGENTS.md", "POST publish"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %#v, want %#v", *calls, want)
	}
}

func TestPushApprovalRequiredSubmitsForReviewInDefaultMode(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{approvalRequired: true})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	stdout := captureStdout(t, func() error {
		return RunPush(nil, true)
	})
	var out map[string]string
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode push JSON %q: %v", stdout, err)
	}
	if out["lifecycle"] != "review" || out["reviewUrl"] != server.URL+"/reviews?package=nwks%2Fplatform%2Fagent-defaults&version=1.0.0" {
		t.Fatalf("push JSON = %+v", out)
	}

	want := []string{"GET package", "POST draft 1.0.0", "PUT Trovefile", "PUT AGENTS.md", "POST publish", "POST submit"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %#v, want %#v", *calls, want)
	}
}

func TestPushPublishModeDoesNotSubmitWhenApprovalRequired(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{approvalRequired: true})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	err := RunPush([]string{"--publish"}, false)
	if err == nil || !strings.Contains(err.Error(), "APPROVAL_REQUIRED") {
		t.Fatalf("RunPush(--publish) error = %v, want APPROVAL_REQUIRED", err)
	}
	for _, call := range *calls {
		if call == "POST submit" {
			t.Fatalf("--publish submitted for review: %#v", *calls)
		}
	}
}

func TestPushReusesExistingHiddenDraft(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{versions: []PackageVersion{}, createConflictLifecycle: "draft"})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	if err := RunPush(nil, false); err != nil {
		t.Fatalf("RunPush() error = %v", err)
	}

	want := []string{"GET package", "POST draft 1.0.0", "GET version 1.0.0", "PUT Trovefile", "PUT AGENTS.md", "POST publish"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %#v, want %#v", *calls, want)
	}
}

func TestPushExistingReviewRequiresForce(t *testing.T) {
	server, _ := newPushServer(t, pushServerOptions{versions: []PackageVersion{}, createConflictLifecycle: "review"})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	err := RunPush(nil, false)
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("RunPush() error = %v, want --force guidance", err)
	}
}

func TestPushForceResetsExistingReview(t *testing.T) {
	server, calls := newPushServer(t, pushServerOptions{versions: []PackageVersion{}, createConflictLifecycle: "review"})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	if err := RunPush([]string{"--force"}, false); err != nil {
		t.Fatalf("RunPush(--force) error = %v", err)
	}

	want := []string{"GET package", "POST draft 1.0.0", "GET version 1.0.0", "POST reset", "PUT Trovefile", "PUT AGENTS.md", "POST publish"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %#v, want %#v", *calls, want)
	}
}

func TestPushForceDoesNotOverwritePublishedVersion(t *testing.T) {
	server, _ := newPushServer(t, pushServerOptions{versions: []PackageVersion{{Version: "1.0.0", Lifecycle: "published"}}})
	setupPushWorktree(t, server.URL)
	t.Setenv("TROVE_SERVER_URL", server.URL)
	t.Setenv("TROVE_TOKEN", "test-token")

	err := RunPush([]string{"--force", "--version", "1.0.0"}, false)
	if err == nil || !strings.Contains(err.Error(), "immutable lifecycle published") {
		t.Fatalf("RunPush(--force --version 1.0.0) error = %v, want immutable lifecycle", err)
	}
}

type pushServerOptions struct {
	versions                []PackageVersion
	approvalRequired        bool
	createConflictLifecycle string
}

func setupPushWorktree(t *testing.T, serverURL string) {
	t.Helper()
	chdirTemp(t)
	m := trovefileWithRemote(RemoteSpec{ServerURL: serverURL, Package: "nwks/platform/agent-defaults", Ref: PackageRef{Org: "nwks", Namespace: "platform", Name: "agent-defaults"}})
	m.Metadata = manifest.Metadata{Org: "nwks", Namespace: "platform", Name: "agent-defaults", DisplayName: "Test", Description: "Test"}
	m.Spec.Artifacts = agentsArtifacts()
	if err := writeTrovefile(manifestPath, m); err != nil {
		t.Fatalf("write trovefile: %v", err)
	}
	if err := os.WriteFile(agentsMDPath, []byte("# Local instructions\n"), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
}

func newPushServer(t *testing.T, opts pushServerOptions) (*httptest.Server, *[]string) {
	t.Helper()
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults":
			calls = append(calls, "GET package")
			if opts.versions == nil {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"code":"PACKAGE_NOT_FOUND","message":"Package was not found.","requestId":"req_pkg"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(PackageDetailResponse{Org: "nwks", Namespace: "platform", Name: "agent-defaults", Visibility: "private", Versions: opts.versions})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/packages":
			calls = append(calls, "POST package")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PackageResponse{Org: "nwks", Namespace: "platform", Name: "agent-defaults", Visibility: "private"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions":
			var req CreateDraftRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode draft request: %v", err)
			}
			calls = append(calls, "POST draft "+req.Version)
			if opts.createConflictLifecycle != "" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":{"code":"VERSION_ALREADY_EXISTS","message":"Package version already exists.","requestId":"req_draft"}}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: req.Version, Lifecycle: "draft"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.0":
			calls = append(calls, "GET version 1.0.0")
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: "1.0.0", Lifecycle: opts.createConflictLifecycle})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.0/artifacts/Trovefile":
			handlePushUpload(t, w, r, &calls, "PUT Trovefile", "apiVersion: trove.io/v1")
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/artifacts/Trovefile":
			handlePushUpload(t, w, r, &calls, "PUT Trovefile", "apiVersion: trove.io/v1")
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/artifacts/AGENTS.md"):
			handlePushUpload(t, w, r, &calls, "PUT AGENTS.md", "# Local instructions\n")
		case r.Method == http.MethodPost && (r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.0/publish" || r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/publish"):
			calls = append(calls, "POST publish")
			if opts.approvalRequired {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":{"code":"APPROVAL_REQUIRED","message":"Version requires review approval before publishing.","requestId":"req_publish"}}`))
				return
			}
			version := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/packages/nwks/platform/agent-defaults/versions/"), "/publish")
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: version, Lifecycle: "published", Digest: "sha256:version"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.0/submit":
			calls = append(calls, "POST submit")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "submitted"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/packages/nwks/platform/agent-defaults/versions/1.0.0/reset-draft":
			calls = append(calls, "POST reset")
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: "1.0.0", Lifecycle: "draft"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func handlePushUpload(t *testing.T, w http.ResponseWriter, r *http.Request, calls *[]string, call string, wantBodyContains string) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read upload body: %v", err)
	}
	if !bytes.Contains(body, []byte(wantBodyContains)) {
		t.Fatalf("%s body %q does not contain %q", call, string(body), wantBodyContains)
	}
	*calls = append(*calls, call)
	_ = json.NewEncoder(w).Encode(ArtifactResponse{Path: strings.TrimPrefix(call, "PUT "), Digest: "sha256:artifact"})
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = writePipe
	err = fn()
	_ = writePipe.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatalf("captured function error = %v", err)
	}
	out, readErr := io.ReadAll(readPipe)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	return string(out)
}
