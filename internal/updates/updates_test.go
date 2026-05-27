package updates

import (
	"encoding/json"
	"strings"
	"testing"

	"trove/internal/testutil"
)

func TestUpdateCheckResponse_JSONShape(t *testing.T) {
	resp := UpdateCheckResponse{
		UpdateAvailable: true,
		LatestVersion:   "1.1.0",
		LatestDigest:    "sha256:def456",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if result["updateAvailable"] != true {
		t.Errorf("updateAvailable = %v, want true", result["updateAvailable"])
	}
	if result["latestVersion"] != "1.1.0" {
		t.Errorf("latestVersion = %v, want 1.1.0", result["latestVersion"])
	}
}

func TestUpdateCheckRequest_InvalidPackage(t *testing.T) {
	service := NewService(nil)

	_, err := service.CheckUpdate(nil, UpdateCheckRequest{
		Package:        "invalid-package",
		CurrentVersion: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error for invalid package format")
	}
	if !strings.Contains(err.Error(), "org/namespace/package") {
		t.Errorf("error = %v, want org/namespace/package format error", err)
	}
}

func TestCheckUpdate_NoUpdateAvailable(t *testing.T) {
	store := testutil.NewPostgresPackageStore(t)
	service := NewService(store)

	resp, err := service.CheckUpdate(nil, UpdateCheckRequest{
		Package:        "companyx/platform/agent-backend",
		CurrentVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("CheckUpdate error = %v", err)
	}
	if resp.UpdateAvailable {
		t.Error("updateAvailable = true, want false")
	}
	if resp.LatestVersion != "1.0.0" {
		t.Errorf("latestVersion = %s, want 1.0.0", resp.LatestVersion)
	}
}
