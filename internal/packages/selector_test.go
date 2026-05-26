package packages

import "testing"

func TestSplitPackageSelector(t *testing.T) {
	name, selector, err := SplitPackageSelector("agent-backend@latest")
	if err != nil {
		t.Fatalf("SplitPackageSelector() error = %v", err)
	}
	if name != "agent-backend" || selector != "latest" {
		t.Fatalf("SplitPackageSelector() = %q, %q", name, selector)
	}
}

func TestParseSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		kind     SelectorKind
	}{
		{name: "latest", selector: "latest", kind: SelectorChannel},
		{name: "exact", selector: "1.2.3", kind: SelectorExact},
		{name: "exact with v", selector: "v1.2.3", kind: SelectorExact},
		{name: "major", selector: "v1", kind: SelectorMajor},
		{name: "minor", selector: "v1.2", kind: SelectorMinor},
		{name: "digest", selector: "sha256:abc", kind: SelectorDigest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSelector(tt.selector)
			if err != nil {
				t.Fatalf("ParseSelector() error = %v", err)
			}
			if parsed.Kind != tt.kind {
				t.Fatalf("kind = %v, want %v", parsed.Kind, tt.kind)
			}
		})
	}
}

func TestParseSelectorRejectsInvalid(t *testing.T) {
	for _, selector := range []string{"", "v", "1.2.3.4", "one"} {
		t.Run(selector, func(t *testing.T) {
			if _, err := ParseSelector(selector); err == nil {
				t.Fatal("ParseSelector() error = nil, want error")
			}
		})
	}
}
