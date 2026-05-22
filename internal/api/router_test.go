package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"trove/internal/config"
)

func TestHealthIncludesRequestID(t *testing.T) {
	router := NewRouter(config.Defaults())
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
	router := NewRouter(config.Defaults())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(HeaderRequestID, "req_client")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if got := res.Header().Get(HeaderRequestID); got != "req_client" {
		t.Fatalf("X-Request-Id = %q, want req_client", got)
	}
}

func TestNotFoundErrorIncludesRequestID(t *testing.T) {
	router := NewRouter(config.Defaults())
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
