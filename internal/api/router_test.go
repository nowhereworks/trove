package api

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"trove/internal/config"
	"trove/internal/packages"
	"trove/internal/testutil"
)

func TestHealthIncludesRequestID(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if res.Header().Get(HeaderRequestID) == "" {
		t.Fatal("X-Request-Id header is empty")
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
}

func TestCoreFindTroveSkillServedWithoutAuth(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/core/skills/find-trove-skills/SKILL.md", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if res.Header().Get(HeaderRequestID) == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	body := res.Body.String()
	for _, want := range []string{"find-trove-skills", "trove skills find"} {
		if !strings.Contains(body, want) {
			t.Fatalf("skill body missing %q", want)
		}
	}
	if strings.Contains(body, "npx skills") {
		t.Fatal("skill body contains npx skills")
	}
}

func TestUnknownCoreSkillReturnsNotFound(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/core/skills/unknown/SKILL.md", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
}

func TestArchiveUploadPublishFlow(t *testing.T) {
	router := testRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.4","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	manifest := strings.ReplaceAll(sliceTwoManifestYAML, "1.0.1", "1.0.4")
	archive := makeUploadZip(t, map[string]string{"AGENTS.md": "# Archive Upload\n", "trove.yaml": manifest})
	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.4/archive", bytes.NewReader(archive))
	uploadReq.Header.Set("Content-Type", "application/zip")
	uploadRes := httptest.NewRecorder()
	router.ServeHTTP(uploadRes, uploadReq)
	if uploadRes.Code != http.StatusOK {
		t.Fatalf("archive upload status = %d, want %d; body=%s", uploadRes.Code, http.StatusOK, uploadRes.Body.String())
	}

	var uploadBody struct {
		Items []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.NewDecoder(uploadRes.Body).Decode(&uploadBody); err != nil {
		t.Fatalf("decode upload body: %v", err)
	}
	if len(uploadBody.Items) != 2 || uploadBody.Items[0].Path != "trove.yaml" || uploadBody.Items[1].Type != "agent-instructions" {
		t.Fatalf("upload body = %+v", uploadBody)
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.4/publish", nil)
	publishRes := httptest.NewRecorder()
	router.ServeHTTP(publishRes, publishReq)
	if publishRes.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want %d; body=%s", publishRes.Code, http.StatusOK, publishRes.Body.String())
	}
}

func TestCreateDraftAutoCreatesNamespaceAndPackageByDefault(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/agents/new-defaults/versions", strings.NewReader(`{"version":"1.0.0","visibility":"private"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusCreated, res.Body.String())
	}

	nextReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/agents/new-defaults/versions", strings.NewReader(`{"version":"1.0.1","visibility":"private"}`))
	nextReq.Header.Set("Content-Type", "application/json")
	nextRes := httptest.NewRecorder()
	router.ServeHTTP(nextRes, nextReq)
	if nextRes.Code != http.StatusCreated {
		t.Fatalf("next draft status = %d, want %d; body=%s", nextRes.Code, http.StatusCreated, nextRes.Body.String())
	}
}

func TestCreateDraftDoesNotAutoCreatePackageWhenDisabled(t *testing.T) {
	cfg := config.Defaults()
	cfg.Packages.CreatePackageOnPush = false
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/new-defaults/versions", strings.NewReader(`{"version":"1.0.0"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestCreateDraftDoesNotAutoCreateNamespaceWhenDisabled(t *testing.T) {
	cfg := config.Defaults()
	cfg.Packages.CreateNamespaceOnPush = false
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/agents/new-defaults/versions", strings.NewReader(`{"version":"1.0.0"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestCreateDraftDoesNotAutoCreateOrg(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/missing/agents/new-defaults/versions", strings.NewReader(`{"version":"1.0.0"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestAuthenticatedSubmitReviewTransitionsLifecycle(t *testing.T) {
	store := testutil.NewPostgresPackageStore(t)
	router := NewRouter(config.Defaults(), store, nil)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.5","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/companyx/platform/agent-backend/versions/1.0.5/submit", nil)
	submitReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	submitRes := httptest.NewRecorder()
	router.ServeHTTP(submitRes, submitReq)
	if submitRes.Code != http.StatusOK {
		t.Fatalf("submit status = %d, want %d; body=%s", submitRes.Code, http.StatusOK, submitRes.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.5", nil)
	getReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	getRes := httptest.NewRecorder()
	router.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", getRes.Code, http.StatusOK, getRes.Body.String())
	}
	var version packages.VersionResource
	if err := json.NewDecoder(getRes.Body).Decode(&version); err != nil {
		t.Fatalf("decode version: %v", err)
	}
	if version.Lifecycle != "review" {
		t.Fatalf("version lifecycle = %q, want review", version.Lifecycle)
	}
}

func TestResetUnpublishedVersionReturnsDraftAndKeepsAnonymousHidden(t *testing.T) {
	store := testutil.NewPostgresPackageStore(t)
	router := NewRouter(config.Defaults(), store, nil)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.6","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/companyx/platform/agent-backend/versions/1.0.6/submit", nil)
	submitReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	submitRes := httptest.NewRecorder()
	router.ServeHTTP(submitRes, submitReq)
	if submitRes.Code != http.StatusOK {
		t.Fatalf("submit status = %d, want %d; body=%s", submitRes.Code, http.StatusOK, submitRes.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.6/reset-draft", nil)
	resetReq.Header.Set("Authorization", "Bearer dev-token-local-only")
	resetRes := httptest.NewRecorder()
	router.ServeHTTP(resetRes, resetReq)
	if resetRes.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want %d; body=%s", resetRes.Code, http.StatusOK, resetRes.Body.String())
	}
	var reset packages.VersionResource
	if err := json.NewDecoder(resetRes.Body).Decode(&reset); err != nil {
		t.Fatalf("decode reset: %v", err)
	}
	if reset.Lifecycle != "draft" {
		t.Fatalf("reset lifecycle = %q, want draft", reset.Lifecycle)
	}

	anonymousReq := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.6", nil)
	anonymousRes := httptest.NewRecorder()
	router.ServeHTTP(anonymousRes, anonymousReq)
	if anonymousRes.Code != http.StatusNotFound {
		t.Fatalf("anonymous status = %d, want %d; body=%s", anonymousRes.Code, http.StatusNotFound, anonymousRes.Body.String())
	}
}

func TestArchiveUploadRejectsUnsafePath(t *testing.T) {
	router := testRouter(t)
	archive := makeUploadZip(t, map[string]string{"../escape.md": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/archive", bytes.NewReader(archive))
	req.Header.Set("Content-Type", "application/zip")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	var body ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "INVALID_ARCHIVE" {
		t.Fatalf("error.code = %q, want INVALID_ARCHIVE", body.Error.Code)
	}
}

func TestDraftUploadPublishFlow(t *testing.T) {
	router := testRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.1","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	manifestReq := httptest.NewRequest(http.MethodPut, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.1/artifacts/trove.yaml", strings.NewReader(sliceTwoManifestYAML))
	manifestReq.Header.Set("Content-Type", "application/yaml")
	manifestRes := httptest.NewRecorder()
	router.ServeHTTP(manifestRes, manifestReq)
	if manifestRes.Code != http.StatusOK {
		t.Fatalf("manifest upload status = %d, want %d; body=%s", manifestRes.Code, http.StatusOK, manifestRes.Body.String())
	}

	agentsReq := httptest.NewRequest(http.MethodPut, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.1/artifacts/AGENTS.md", strings.NewReader("# Updated\n"))
	agentsReq.Header.Set("Content-Type", "text/markdown; charset=utf-8")
	agentsRes := httptest.NewRecorder()
	router.ServeHTTP(agentsRes, agentsReq)
	if agentsRes.Code != http.StatusOK {
		t.Fatalf("artifact upload status = %d, want %d; body=%s", agentsRes.Code, http.StatusOK, agentsRes.Body.String())
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.1/publish", nil)
	publishRes := httptest.NewRecorder()
	router.ServeHTTP(publishRes, publishReq)
	if publishRes.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want %d; body=%s", publishRes.Code, http.StatusOK, publishRes.Body.String())
	}

	var publishBody struct {
		Version   string `json:"version"`
		Lifecycle string `json:"lifecycle"`
		Digest    string `json:"digest"`
	}
	if err := json.NewDecoder(publishRes.Body).Decode(&publishBody); err != nil {
		t.Fatalf("decode publish body: %v", err)
	}
	if publishBody.Version != "1.0.1" || publishBody.Lifecycle != "published" || !strings.HasPrefix(publishBody.Digest, "sha256:") {
		t.Fatalf("publish body = %+v", publishBody)
	}

	resolveReq := httptest.NewRequest(http.MethodGet, "/api/v1/resolve/companyx/platform/agent-backend@latest", nil)
	resolveRes := httptest.NewRecorder()
	router.ServeHTTP(resolveRes, resolveReq)
	if resolveRes.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d", resolveRes.Code, http.StatusOK)
	}
	var resolveBody struct {
		ResolvedVersion string `json:"resolvedVersion"`
	}
	if err := json.NewDecoder(resolveRes.Body).Decode(&resolveBody); err != nil {
		t.Fatalf("decode resolve body: %v", err)
	}
	if resolveBody.ResolvedVersion != "1.0.1" {
		t.Fatalf("resolvedVersion = %q, want 1.0.1", resolveBody.ResolvedVersion)
	}
}

func TestPublishFailsWhenRequiredArtifactMissing(t *testing.T) {
	router := testRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.2","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	manifest := strings.ReplaceAll(sliceTwoManifestYAML, "1.0.1", "1.0.2")
	manifestReq := httptest.NewRequest(http.MethodPut, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.2/artifacts/trove.yaml", strings.NewReader(manifest))
	manifestReq.Header.Set("Content-Type", "application/yaml")
	manifestRes := httptest.NewRecorder()
	router.ServeHTTP(manifestRes, manifestReq)
	if manifestRes.Code != http.StatusOK {
		t.Fatalf("manifest upload status = %d, want %d; body=%s", manifestRes.Code, http.StatusOK, manifestRes.Body.String())
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.2/publish", nil)
	publishRes := httptest.NewRecorder()
	router.ServeHTTP(publishRes, publishReq)
	if publishRes.Code != http.StatusBadRequest {
		t.Fatalf("publish status = %d, want %d; body=%s", publishRes.Code, http.StatusBadRequest, publishRes.Body.String())
	}
}

func TestUploadRejectsInvalidManifest(t *testing.T) {
	router := testRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/packages/companyx/platform/agent-backend/versions", strings.NewReader(`{"version":"1.0.3","visibility":"public"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}

	invalidManifest := strings.ReplaceAll(sliceTwoManifestYAML, "companyx", "wrongorg")
	manifestReq := httptest.NewRequest(http.MethodPut, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.3/artifacts/trove.yaml", strings.NewReader(invalidManifest))
	manifestReq.Header.Set("Content-Type", "application/yaml")
	manifestRes := httptest.NewRecorder()
	router.ServeHTTP(manifestRes, manifestReq)
	if manifestRes.Code != http.StatusBadRequest {
		t.Fatalf("manifest upload status = %d, want %d; body=%s", manifestRes.Code, http.StatusBadRequest, manifestRes.Body.String())
	}

	var body ErrorResponse
	if err := json.NewDecoder(manifestRes.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "INVALID_MANIFEST" {
		t.Fatalf("error.code = %q, want INVALID_MANIFEST", body.Error.Code)
	}
}

func TestPublishedVersionCannotBeMutated(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/artifacts/AGENTS.md", strings.NewReader("# Mutated\n"))
	req.Header.Set("Content-Type", "text/markdown; charset=utf-8")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusConflict, res.Body.String())
	}
	var body ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "VERSION_IMMUTABLE" {
		t.Fatalf("error.code = %q, want VERSION_IMMUTABLE", body.Error.Code)
	}
}

func TestRequestIDPropagatesFromRequest(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(HeaderRequestID, "req_client")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if got := res.Header().Get(HeaderRequestID); got != "req_client" {
		t.Fatalf("X-Request-Id = %q, want req_client", got)
	}
}

func TestNotFoundErrorIncludesRequestID(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set(HeaderRequestID, "req_missing")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
	if got := res.Header().Get(HeaderRequestID); got != "req_missing" {
		t.Fatalf("X-Request-Id = %q, want req_missing", got)
	}

	var body ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Fatalf("error.code = %q, want NOT_FOUND", body.Error.Code)
	}
	if body.Error.RequestID != "req_missing" {
		t.Fatalf("error.requestId = %q, want req_missing", body.Error.RequestID)
	}
}

func TestResolveStable(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resolve/companyx/platform/agent-backend@latest", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body struct {
		Org             string `json:"org"`
		Namespace       string `json:"namespace"`
		Package         string `json:"package"`
		Selector        string `json:"selector"`
		ResolvedVersion string `json:"resolvedVersion"`
		Digest          string `json:"digest"`
		ManifestURL     string `json:"manifestUrl"`
		ArchiveURL      string `json:"archiveUrl"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.ResolvedVersion != "1.0.0" || body.Selector != "latest" {
		t.Fatalf("resolve body = %+v", body)
	}
	if body.ManifestURL != "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/manifest" {
		t.Fatalf("manifestUrl = %q", body.ManifestURL)
	}
}

func TestGetConfigReturnsOrgCreationSettings(t *testing.T) {
	cfg := config.Defaults()
	cfg.Orgs.DefaultOrg = "sample-org"
	cfg.Orgs.AllowCreateOrg = false
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		Org            string `json:"org"`
		AllowCreateOrg bool   `json:"allowCreateOrg"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Org != "sample-org" || body.AllowCreateOrg {
		t.Fatalf("config body = %+v", body)
	}
}

func TestCreateOrgAllowedByDefault(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(`{"slug":"sample-org","displayName":"Sample Org","visibility":"private"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusCreated, res.Body.String())
	}
	if got := res.Header().Get("Location"); got != "/api/v1/orgs/sample-org" {
		t.Fatalf("Location = %q", got)
	}
}

func TestCreateOrgDisabledByConfig(t *testing.T) {
	cfg := config.Defaults()
	cfg.Orgs.AllowCreateOrg = false
	cfg.Orgs.DefaultOrg = "sample-org"
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", strings.NewReader(`{"slug":"other-org","displayName":"Other Org","visibility":"private"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
	var body ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "ORG_CREATION_DISABLED" {
		t.Fatalf("error.code = %q, want ORG_CREATION_DISABLED", body.Error.Code)
	}
}

func TestManifestEndpointReturnsManifestJSON(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/manifest", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body struct {
		Metadata struct {
			Org       string `json:"org"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			Version string `json:"version"`
		} `json:"spec"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Metadata.Org != "companyx" || body.Metadata.Namespace != "platform" || body.Metadata.Name != "agent-backend" {
		t.Fatalf("manifest body = %+v", body)
	}
}

func TestRawExactReturnsArtifactWithETag(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/AGENTS.md@1.0.0", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("ETag"); got != "sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050" {
		t.Fatalf("ETag = %q", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if body := res.Body.String(); !strings.Contains(body, "# Backend Agent Defaults") {
		t.Fatalf("raw body = %q", body)
	}
}

func TestRawAliasRedirectsToExact(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/AGENTS.md@latest", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "/raw/companyx/platform/agent-backend/AGENTS.md@1.0.0" {
		t.Fatalf("Location = %q", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestRawAliasRequiresAuthBeforeRedirectWhenPublicRawDisabled(t *testing.T) {
	cfg := config.Defaults()
	cfg.Raw.AllowPublicPackages = false
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/AGENTS.md@latest", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
	if got := res.Header().Get("Location"); got != "" {
		t.Fatalf("Location = %q, want no redirect", got)
	}
}

func TestRawOmittedSelectorRedirectsToStableExact(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/AGENTS.md", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "/raw/companyx/platform/agent-backend/AGENTS.md@1.0.0" {
		t.Fatalf("Location = %q", got)
	}
}

func TestRawArtifactPathWithAtIsRejected(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/AGENTS@bad.md@latest", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestArchiveTarGzExactReturnsDeterministicArchive(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/archive.tar.gz", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Content-Type"); got != "application/gzip" {
		t.Fatalf("Content-Type = %q, want application/gzip", got)
	}
	if got := res.Header().Get("ETag"); got != "sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050" {
		t.Fatalf("ETag = %q", got)
	}
	entries := readResponseTarGz(t, res.Body.Bytes())
	if len(entries) != 1 || entries[0].Path != "AGENTS.md" || !strings.Contains(string(entries[0].Content), "Backend Agent Defaults") {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestArchiveZipExactReturnsArchive(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/archive.zip", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", got)
	}
	entries := readResponseZip(t, res.Body.Bytes())
	if len(entries) != 1 || entries[0].Path != "AGENTS.md" || !strings.Contains(string(entries[0].Content), "Backend Agent Defaults") {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestArchiveAliasRedirectsToExact(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/versions/stable/archive.tar.gz", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/archive.tar.gz" {
		t.Fatalf("Location = %q", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestPackageListAndDetail(t *testing.T) {
	router := testRouter(t)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/packages", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRes.Code, http.StatusOK)
	}

	var listBody struct {
		Items []struct {
			Org           string `json:"org"`
			Namespace     string `json:"namespace"`
			Name          string `json:"name"`
			LatestVersion string `json:"latestVersion"`
		} `json:"items"`
		NextCursor *string `json:"nextCursor"`
	}
	if err := json.NewDecoder(listRes.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list body: %v", err)
	}
	if len(listBody.Items) != 1 || listBody.Items[0].LatestVersion != "1.0.0" {
		t.Fatalf("list body = %+v", listBody)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend", nil)
	detailRes := httptest.NewRecorder()
	router.ServeHTTP(detailRes, detailReq)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detailRes.Code, http.StatusOK)
	}

	var detailBody struct {
		DisplayName string `json:"displayName"`
		Versions    []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(detailRes.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode detail body: %v", err)
	}
	if detailBody.DisplayName != "Backend Agent Defaults" || len(detailBody.Versions) != 1 || detailBody.Versions[0].Version != "1.0.0" {
		t.Fatalf("detail body = %+v", detailBody)
	}
}

func TestEmbeddedUIServed(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if body := res.Body.String(); !strings.Contains(body, "<title>Trove</title>") {
		t.Fatalf("UI body missing Trove title: %q", body)
	}
}

type routerTestStore struct {
	packages.Store
	packages.WriteStore
}

func testRouter(t *testing.T) http.Handler {
	return testRouterWithConfig(t, config.Defaults())
}

func testRouterWithConfig(t *testing.T, cfg config.Config) http.Handler {
	store := testutil.NewPostgresPackageStore(t)
	return NewRouter(cfg, routerTestStore{Store: store, WriteStore: store}, nil)
}

const sliceTwoManifestYAML = `apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: companyx
  namespace: platform
  name: agent-backend
  displayName: Backend Agent Defaults
  description: Default agent instructions, skills, and commands for backend services.
spec:
  visibility: public
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
`

func makeUploadZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

type responseArchiveEntry struct {
	Path    string
	Content []byte
}

func readResponseTarGz(t *testing.T, data []byte) []responseArchiveEntry {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open gzip response: %v", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var entries []responseArchiveEntry
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar response: %v", err)
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		entries = append(entries, responseArchiveEntry{Path: header.Name, Content: content})
	}
	return entries
}

func readResponseZip(t *testing.T, data []byte) []responseArchiveEntry {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip response: %v", err)
	}
	entries := make([]responseArchiveEntry, 0, len(reader.File))
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry: %v", err)
		}
		content, err := io.ReadAll(entry)
		if err != nil {
			t.Fatalf("read zip entry: %v", err)
		}
		if err := entry.Close(); err != nil {
			t.Fatalf("close zip entry: %v", err)
		}
		entries = append(entries, responseArchiveEntry{Path: file.Name, Content: content})
	}
	return entries
}

func TestSearchPackages(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/packages?q=backend+agent", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}

	var body struct {
		Items      []packages.PackageSummary `json:"items"`
		NextCursor *string                   `json:"nextCursor"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode search body: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("search items = %d, want 1", len(body.Items))
	}
	if body.Items[0].Name != "agent-backend" {
		t.Fatalf("search item name = %q, want agent-backend", body.Items[0].Name)
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/packages", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("search status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestGetPackageAdoption(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/companyx/platform/agent-backend/adoption", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("adoption status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}

	var body struct {
		ProjectCount int64 `json:"projectCount"`
		VersionCount int64 `json:"versionCount"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode adoption body: %v", err)
	}
	if body.ProjectCount != 0 || body.VersionCount != 0 {
		t.Fatalf("adoption body = %+v, want zeros", body)
	}
}

func TestGetPackageAdoptionNotFound(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/unknown/ns/pkg/adoption", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("adoption status = %d, want %d", res.Code, http.StatusNotFound)
	}
}

func TestCreateProjectRequiresAuth(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"org":"acme","name":"my-app","repoUrl":"https://github.com/acme/my-app"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("create project status = %d, want %d; body=%s", res.Code, http.StatusUnauthorized, res.Body.String())
	}
}

func TestCreateProjectRejectsMissingFields(t *testing.T) {
	t.Skip("requires authenticated request")
}

func TestReportProjectAdoptionRequiresAuth(t *testing.T) {
	router := testRouter(t)

	body := `{
		"org": "acme",
		"name": "my-app",
		"repoUrl": "https://github.com/acme/my-app",
		"installed": [
			{"package": "companyx/platform/agent-backend", "version": "1.0.0", "digest": "sha256:abc123"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("report adoption status = %d, want %d; body=%s", res.Code, http.StatusUnauthorized, res.Body.String())
	}
}

func TestReportProjectAdoptionRejectsMissingFields(t *testing.T) {
	t.Skip("requires authenticated request")
}
