package updates

import (
	"encoding/json"
	"strings"
	"testing"

	"trove/internal/packages"
)

func TestSatisfiesSemVer_Equality(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "=1.0.0", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "!=1.0.0", false},
		{"1.0.1", "!=1.0.0", true},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_Comparisons(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.1", ">1.0.0", true},
		{"1.0.0", ">1.0.0", false},
		{"1.0.0", ">=1.0.0", true},
		{"0.9.9", ">=1.0.0", false},
		{"1.0.0", "<2.0.0", true},
		{"2.0.0", "<2.0.0", false},
		{"1.9.9", "<=2.0.0", true},
		{"2.0.1", "<=2.0.0", false},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_RangeConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"0.6.0", ">=0.6.0 <2.0.0", true},
		{"1.5.0", ">=0.6.0 <2.0.0", true},
		{"1.9.9", ">=0.6.0 <2.0.0", true},
		{"0.5.9", ">=0.6.0 <2.0.0", false},
		{"2.0.0", ">=0.6.0 <2.0.0", false},
		{"1.0.0", ">=1.0.0, <2.0.0", true},
		{"1.5.0", ">=1.0.0, <2.0.0", true},
		{"2.0.0", ">=1.0.0, <2.0.0", false},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_CaretConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.0", "^1.0.0", true},
		{"1.5.0", "^1.0.0", true},
		{"1.9.9", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"0.9.9", "^1.0.0", false},
		{"0.1.0", "^0.1.0", true},
		{"0.1.5", "^0.1.0", true},
		{"0.2.0", "^0.1.0", false},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_TildeConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.2.0", "~1.2.0", true},
		{"1.2.5", "~1.2.0", true},
		{"1.3.0", "~1.2.0", false},
		{"1.1.9", "~1.2.0", false},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_WildcardConstraints(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.0", "1.x", true},
		{"1.5.0", "1.x", true},
		{"2.0.0", "1.x", false},
		{"1.2.0", "1.2.x", true},
		{"1.2.5", "1.2.x", true},
		{"1.3.0", "1.2.x", false},
		{"1.0.0", "*", true},
		{"99.0.0", "*", true},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_HyphenRange(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.0", "1.0.0 - 2.0.0", true},
		{"1.5.0", "1.0.0 - 2.0.0", true},
		{"2.0.0", "1.0.0 - 2.0.0", true},
		{"0.9.9", "1.0.0 - 2.0.0", false},
		{"2.0.1", "1.0.0 - 2.0.0", false},
	}
	for _, tt := range tests {
		got, err := satisfiesSemVer(tt.version, tt.constraint)
		if err != nil {
			t.Fatalf("satisfiesSemVer(%q, %q) error = %v", tt.version, tt.constraint, err)
		}
		if got != tt.want {
			t.Errorf("satisfiesSemVer(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestSatisfiesSemVer_EmptyConstraint(t *testing.T) {
	got, err := satisfiesSemVer("1.0.0", "")
	if err != nil {
		t.Fatalf("satisfiesSemVer error = %v", err)
	}
	if !got {
		t.Error("empty constraint should always satisfy")
	}
}

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"0.9.9", "1.0.0", -1},
	}
	for _, tt := range tests {
		aMajor, aMinor, aPatch, _ := packages.ParseStrictSemver(tt.a)
		bMajor, bMinor, bPatch, _ := packages.ParseStrictSemver(tt.b)
		got := compareSemVer(aMajor, aMinor, aPatch, bMajor, bMinor, bPatch)
		if got != tt.want {
			t.Errorf("compareSemVer(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestUpdateCheckResponse_JSONShape(t *testing.T) {
	resp := UpdateCheckResponse{
		UpdateAvailable:        true,
		LatestVersion:          "1.1.0",
		LatestDigest:           "sha256:def456",
		Compatibility:          "compatible",
		RequiresManualApproval: false,
		ChangelogURL:           "/api/v1/packages/companyx/platform/agent-backend/compare/1.0.0...1.1.0",
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
	if result["compatibility"] != "compatible" {
		t.Errorf("compatibility = %v, want compatible", result["compatibility"])
	}
	if result["changelogUrl"] != "/api/v1/packages/companyx/platform/agent-backend/compare/1.0.0...1.1.0" {
		t.Errorf("changelogUrl = %v", result["changelogUrl"])
	}
}

func TestCompatibilityCheckResponse_JSONShape(t *testing.T) {
	resp := CompatibilityCheckResponse{
		Compatibility: "compatible",
		Details: []CompatibilityDetail{
			{Kind: "tool", Name: "opencode", Required: ">=0.6.0 <2.0.0", Provided: "0.6.0", Compatible: true},
			{Kind: "runtime", Name: "linux", Required: "linux", Provided: "linux", Compatible: true},
			{Kind: "model", Name: "gpt", Required: "gpt (min 128000 context)", Provided: "gpt (128000 context)", Compatible: true},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if result["compatibility"] != "compatible" {
		t.Errorf("compatibility = %v, want compatible", result["compatibility"])
	}

	details, ok := result["details"].([]any)
	if !ok || len(details) != 3 {
		t.Fatalf("details length = %d, want 3", len(details))
	}

	first := details[0].(map[string]any)
	if first["kind"] != "tool" || first["compatible"] != true {
		t.Errorf("first detail = %+v", first)
	}
}

func TestUpdateCheckRequest_InvalidPackage(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

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

func TestCompatibilityCheckRequest_InvalidPackage(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	_, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "invalid",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error for invalid package format")
	}
}

func TestCheckCompatibility_ToolCompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			Tool:        "opencode",
			ToolVersion: "0.6.0",
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "compatible" {
		t.Errorf("compatibility = %s, want compatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_ToolIncompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			Tool:        "opencode",
			ToolVersion: "2.0.0",
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_RuntimeCompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			Runtime: "linux",
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "compatible" {
		t.Errorf("compatibility = %s, want compatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_RuntimeIncompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			Runtime: "windows",
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_ModelCompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			ModelFamily:   "gpt",
			ContextWindow: 128000,
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "compatible" {
		t.Errorf("compatibility = %s, want compatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_ModelInsufficientContext(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target: Target{
			ModelFamily:   "gpt",
			ContextWindow: 64000,
		},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "incompatible" {
		t.Errorf("compatibility = %s, want incompatible", resp.Compatibility)
	}
}

func TestCheckCompatibility_NoTargetReturnsCompatible(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckCompatibility(nil, CompatibilityCheckRequest{
		Package: "companyx/platform/agent-backend",
		Version: "1.0.0",
		Target:  Target{},
	})
	if err != nil {
		t.Fatalf("CheckCompatibility error = %v", err)
	}
	if resp.Compatibility != "compatible" {
		t.Errorf("compatibility = %s, want compatible", resp.Compatibility)
	}
}

func TestCheckUpdate_NoUpdateAvailable(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckUpdate(nil, UpdateCheckRequest{
		Package:        "companyx/platform/agent-backend",
		CurrentVersion: "1.0.0",
		Channel:        "stable",
		Target: Target{
			Tool:        "opencode",
			ToolVersion: "0.6.0",
		},
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

func TestCheckUpdate_ChangelogURL(t *testing.T) {
	store := packages.NewSeedMemoryStore()
	service := NewService(store)

	resp, err := service.CheckUpdate(nil, UpdateCheckRequest{
		Package:        "companyx/platform/agent-backend",
		CurrentVersion: "1.0.0",
		Channel:        "stable",
	})
	if err != nil {
		t.Fatalf("CheckUpdate error = %v", err)
	}
	expected := "/api/v1/packages/companyx/platform/agent-backend/compare/1.0.0...1.0.0"
	if resp.ChangelogURL != expected {
		t.Errorf("changelogUrl = %s, want %s", resp.ChangelogURL, expected)
	}
}
