package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestClientWriteMethods(t *testing.T) {
	t.Setenv("TROVE_TOKEN", "test-token")
	var calls []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/packages/nwks/platform/agent-defaults":
			_ = json.NewEncoder(w).Encode(PackageDetailResponse{Org: "nwks", Namespace: "platform", Name: "agent-defaults", Visibility: "private", Versions: []PackageVersion{{Version: "1.0.0", Lifecycle: "published"}}})
		case "POST /api/v1/packages":
			var req CreatePackageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create package body: %v", err)
			}
			if req.Org != "nwks" || req.Namespace != "platform" || req.Name != "agent-defaults" || req.Visibility != "private" {
				t.Fatalf("create package body = %+v", req)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PackageResponse{Org: req.Org, Namespace: req.Namespace, Name: req.Name, Visibility: req.Visibility})
		case "GET /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1":
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: "1.0.1", Lifecycle: "draft", Visibility: "private"})
		case "POST /api/v1/packages/nwks/platform/agent-defaults/versions":
			var req CreateDraftRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create draft body: %v", err)
			}
			if req.Version != "1.0.1" || req.Visibility != "private" {
				t.Fatalf("create draft body = %+v", req)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: req.Version, Lifecycle: "draft", Visibility: req.Visibility})
		case "PUT /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/artifacts/AGENTS.md":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			if string(body) != "# Agents\n" || r.Header.Get("Content-Type") != "text/markdown" {
				t.Fatalf("upload body/header = %q/%q", string(body), r.Header.Get("Content-Type"))
			}
			_ = json.NewEncoder(w).Encode(ArtifactResponse{Path: "AGENTS.md", Type: "agent-instructions", Digest: "sha256:artifact", SizeBytes: int64(len(body)), TargetPath: "AGENTS.md"})
		case "POST /api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.1/submit":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "submitted"})
		case "GET /api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.1/approval-status":
			_ = json.NewEncoder(w).Encode(ApprovalStatusResponse{HasEnoughApprovals: true, CurrentCount: 1, RequiredCount: 1})
		case "POST /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/publish":
			_ = json.NewEncoder(w).Encode(VersionResponse{Org: "nwks", Namespace: "platform", Package: "agent-defaults", Version: "1.0.1", Lifecycle: "published", Visibility: "private", Digest: "sha256:version"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClientForServer(server.URL)
	if detail, err := client.GetPackage("nwks", "platform", "agent-defaults"); err != nil || len(detail.Versions) != 1 {
		t.Fatalf("GetPackage() = %+v, %v", detail, err)
	}
	if pkg, err := client.CreatePackage(CreatePackageRequest{Org: "nwks", Namespace: "platform", Name: "agent-defaults", DisplayName: "Agent Defaults", Description: "Shared AGENTS.md instructions.", Visibility: "private"}); err != nil || pkg.Name != "agent-defaults" {
		t.Fatalf("CreatePackage() = %+v, %v", pkg, err)
	}
	if version, err := client.GetVersion("nwks", "platform", "agent-defaults", "1.0.1"); err != nil || version.Lifecycle != "draft" {
		t.Fatalf("GetVersion() = %+v, %v", version, err)
	}
	if draft, err := client.CreateDraft("nwks", "platform", "agent-defaults", CreateDraftRequest{Version: "1.0.1", Visibility: "private"}); err != nil || draft.Version != "1.0.1" {
		t.Fatalf("CreateDraft() = %+v, %v", draft, err)
	}
	if artifact, err := client.UploadArtifact("nwks", "platform", "agent-defaults", "1.0.1", "AGENTS.md", []byte("# Agents\n"), "text/markdown"); err != nil || artifact.Digest != "sha256:artifact" {
		t.Fatalf("UploadArtifact() = %+v, %v", artifact, err)
	}
	if err := client.SubmitReview("nwks", "platform", "agent-defaults", "1.0.1"); err != nil {
		t.Fatalf("SubmitReview() error = %v", err)
	}
	if status, err := client.ApprovalStatus("nwks", "platform", "agent-defaults", "1.0.1"); err != nil || !status.HasEnoughApprovals {
		t.Fatalf("ApprovalStatus() = %+v, %v", status, err)
	}
	if published, err := client.PublishVersion("nwks", "platform", "agent-defaults", "1.0.1"); err != nil || published.Lifecycle != "published" {
		t.Fatalf("PublishVersion() = %+v, %v", published, err)
	}

	wantCalls := []string{
		"GET /api/v1/packages/nwks/platform/agent-defaults",
		"POST /api/v1/packages",
		"GET /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1",
		"POST /api/v1/packages/nwks/platform/agent-defaults/versions",
		"PUT /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/artifacts/AGENTS.md",
		"POST /api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.1/submit",
		"GET /api/v1/reviews/nwks/platform/agent-defaults/versions/1.0.1/approval-status",
		"POST /api/v1/packages/nwks/platform/agent-defaults/versions/1.0.1/publish",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestClientStructuredAPIError(t *testing.T) {
	t.Setenv("TROVE_TOKEN", "test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_header")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"APPROVAL_REQUIRED","message":"Version requires review approval before publishing.","requestId":"req_body","details":{"requiredApprovals":1}}}`))
	}))
	t.Cleanup(server.Close)

	_, err := NewClientForServer(server.URL).PublishVersion("nwks", "platform", "agent-defaults", "1.0.1")
	if err == nil {
		t.Fatal("PublishVersion() error = nil, want API error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusForbidden || apiErr.Code != "APPROVAL_REQUIRED" || apiErr.RequestID != "req_body" || apiErr.Details["requiredApprovals"] != float64(1) {
		t.Fatalf("APIError = %+v", apiErr)
	}
	if !strings.Contains(apiErr.Error(), "request req_body") {
		t.Fatalf("APIError string = %q", apiErr.Error())
	}
}

func TestClientStructuredAPIErrorUsesHeaderRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_header")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"VERSION_ALREADY_EXISTS","message":"Version already exists."}}`))
	}))
	t.Cleanup(server.Close)

	_, err := NewClientForServer(server.URL).CreateDraft("nwks", "platform", "agent-defaults", CreateDraftRequest{Version: "1.0.1", Visibility: "private"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusConflict || apiErr.Code != "VERSION_ALREADY_EXISTS" || apiErr.RequestID != "req_header" {
		t.Fatalf("APIError = %+v", apiErr)
	}
}

func TestClientSearchPackages(t *testing.T) {
	nextCursor := "cursor-2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/search/packages" {
			t.Fatalf("request = %s %s, want GET /api/v1/search/packages", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "react performance" {
			t.Fatalf("q = %q, want react performance", got)
		}
		if got := r.URL.Query().Get("artifactType"); got != "skill" {
			t.Fatalf("artifactType = %q, want skill", got)
		}
		_ = json.NewEncoder(w).Encode(SearchPackagesResponse{
			Items: []PackageSummary{{
				Org:           "nwks",
				Namespace:     "platform",
				Name:          "react-best-practices",
				Description:   "React and Next.js performance optimization guidelines.",
				StableVersion: "1.0.0",
			}},
			NextCursor: &nextCursor,
		})
	}))
	t.Cleanup(server.Close)

	result, err := NewClientForServer(server.URL).SearchPackages(SearchPackagesParams{Query: "react performance", ArtifactType: "skill"})
	if err != nil {
		t.Fatalf("SearchPackages() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "react-best-practices" || result.NextCursor == nil || *result.NextCursor != nextCursor {
		t.Fatalf("SearchPackages() = %+v", result)
	}
}
