package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateCheck_ReturnsUpdateAvailable(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package":        "companyx/platform/agent-backend",
		"currentVersion": "0.9.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}

	var resp struct {
		UpdateAvailable bool   `json:"updateAvailable"`
		LatestVersion   string `json:"latestVersion"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp.UpdateAvailable {
		t.Error("updateAvailable = false, want true")
	}
	if resp.LatestVersion != "1.0.0" {
		t.Errorf("latestVersion = %s, want 1.0.0", resp.LatestVersion)
	}
}

func TestUpdateCheck_NoUpdateWhenCurrentIsLatest(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package":        "companyx/platform/agent-backend",
		"currentVersion": "1.0.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var resp struct {
		UpdateAvailable bool `json:"updateAvailable"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.UpdateAvailable {
		t.Error("updateAvailable = true, want false")
	}
}

func TestUpdateCheck_RejectsInvalidJSON(t *testing.T) {
	router := testRouterWithUpdates(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Error.Code != "INVALID_JSON" {
		t.Errorf("error.code = %s, want INVALID_JSON", resp.Error.Code)
	}
}

func TestUpdateCheck_RejectsMissingPackage(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"currentVersion": "1.0.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestUpdateCheck_RejectsMissingCurrentVersion(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestUpdateCheck_IncludesRequestID(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package":        "companyx/platform/agent-backend",
		"currentVersion": "1.0.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/updates/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderRequestID, "req_test_update")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Header().Get(HeaderRequestID) != "req_test_update" {
		t.Errorf("X-Request-Id = %s, want req_test_update", res.Header().Get(HeaderRequestID))
	}
}

func testRouterWithUpdates(t *testing.T) http.Handler {
	return testRouter(t)
}
