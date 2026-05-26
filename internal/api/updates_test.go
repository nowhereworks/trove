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
		"channel":        "stable",
		"target": map[string]any{
			"tool":        "opencode",
			"toolVersion": "0.6.0",
		},
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
		Compatibility   string `json:"compatibility"`
		ChangelogURL    string `json:"changelogUrl"`
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
	if !strings.Contains(resp.ChangelogURL, "compare/0.9.0...1.0.0") {
		t.Errorf("changelogUrl = %s, want compare/0.9.0...1.0.0", resp.ChangelogURL)
	}
}

func TestUpdateCheck_NoUpdateWhenCurrentIsLatest(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package":        "companyx/platform/agent-backend",
		"currentVersion": "1.0.0",
		"channel":        "stable",
		"target": map[string]any{
			"tool":        "opencode",
			"toolVersion": "0.6.0",
		},
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

func TestCompatibilityCheck_Compatible(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
		"version": "1.0.0",
		"target": map[string]any{
			"tool":          "opencode",
			"toolVersion":   "0.6.0",
			"runtime":       "linux",
			"modelFamily":   "gpt",
			"contextWindow": 128000,
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}

	var resp struct {
		Compatibility string `json:"compatibility"`
		Details       []struct {
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			Compatible bool   `json:"compatible"`
		} `json:"details"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Compatibility != "compatible" {
		t.Errorf("compatibility = %s, want compatible", resp.Compatibility)
	}
	if len(resp.Details) == 0 {
		t.Error("expected compatibility details")
	}
}

func TestCompatibilityCheck_IncompatibleTool(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
		"version": "1.0.0",
		"target": map[string]any{
			"tool":        "opencode",
			"toolVersion": "2.0.0",
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var resp struct {
		Compatibility string `json:"compatibility"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCompatibilityCheck_IncompatibleRuntime(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
		"version": "1.0.0",
		"target": map[string]any{
			"runtime": "windows",
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var resp struct {
		Compatibility string `json:"compatibility"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCompatibilityCheck_IncompatibleModel(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
		"version": "1.0.0",
		"target": map[string]any{
			"modelFamily":   "gpt",
			"contextWindow": 64000,
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var resp struct {
		Compatibility string `json:"compatibility"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCompatibilityCheck_RejectsInvalidJSON(t *testing.T) {
	router := testRouterWithUpdates(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestCompatibilityCheck_RejectsMissingPackage(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"version": "1.0.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestCompatibilityCheck_RejectsMissingVersion(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "companyx/platform/agent-backend",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestCompatibilityCheck_UnknownPackage(t *testing.T) {
	router := testRouterWithUpdates(t)

	reqBody := map[string]any{
		"package": "unknown/ns/pkg",
		"version": "1.0.0",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
}

func testRouterWithUpdates(t *testing.T) http.Handler {
	return testRouter(t)
}
