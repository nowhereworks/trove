package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"trove/internal/config"
	"trove/internal/packages"
)

func TestHealthIncludesRequestID(t *testing.T) {
	router := testRouter()
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

func TestRequestIDPropagatesFromRequest(t *testing.T) {
	router := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(HeaderRequestID, "req_client")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if got := res.Header().Get(HeaderRequestID); got != "req_client" {
		t.Fatalf("X-Request-Id = %q, want req_client", got)
	}
}

func TestNotFoundErrorIncludesRequestID(t *testing.T) {
	router := testRouter()
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
	router := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resolve/companyx/platform/agent-backend@stable", nil)
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
	if body.ResolvedVersion != "1.0.0" || body.Selector != "stable" {
		t.Fatalf("resolve body = %+v", body)
	}
	if body.ManifestURL != "/api/v1/packages/companyx/platform/agent-backend/versions/1.0.0/manifest" {
		t.Fatalf("manifestUrl = %q", body.ManifestURL)
	}
}

func TestManifestEndpointReturnsManifestJSON(t *testing.T) {
	router := testRouter()
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
	if body.Metadata.Org != "companyx" || body.Metadata.Namespace != "platform" || body.Metadata.Name != "agent-backend" || body.Spec.Version != "1.0.0" {
		t.Fatalf("manifest body = %+v", body)
	}
}

func TestRawExactReturnsArtifactWithETag(t *testing.T) {
	router := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/1.0.0/AGENTS.md", nil)
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
	router := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/raw/companyx/platform/agent-backend/stable/AGENTS.md", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "/raw/companyx/platform/agent-backend/1.0.0/AGENTS.md" {
		t.Fatalf("Location = %q", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestPackageListAndDetail(t *testing.T) {
	router := testRouter()

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/search/packages", nil)
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
			StableVersion string `json:"stableVersion"`
		} `json:"items"`
		NextCursor *string `json:"nextCursor"`
	}
	if err := json.NewDecoder(listRes.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list body: %v", err)
	}
	if len(listBody.Items) != 1 || listBody.Items[0].StableVersion != "1.0.0" {
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
	router := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if body := res.Body.String(); !strings.Contains(body, "Trove Registry") {
		t.Fatalf("UI body missing Trove Registry: %q", body)
	}
}

func testRouter() http.Handler {
	return NewRouter(config.Defaults(), packages.NewSeedMemoryStore(), nil)
}
