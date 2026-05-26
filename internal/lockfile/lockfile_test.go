package lockfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseValidLockFile(t *testing.T) {
	content := []byte(`apiVersion: trove.io/v1
kind: TroveLock
generatedBy:
  name: trove
  version: 0.1.0
  generatedAt: "2026-05-21T00:00:00Z"
project:
  org: companyx
  name: payments-api
  repo: https://git.company.com/payments/payments-api
installs:
  - package: companyx/platform/agent-backend
    requestedSelector: latest
    version: 1.0.0
    digest: sha256:abc123
    installedAt: "2026-05-21T00:00:00Z"
    artifacts:
      - source: AGENTS.md
        target: AGENTS.md
        digest: sha256:def456
`)

	lock, err := Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if lock.APIVersion != LockAPIVersion {
		t.Errorf("apiVersion = %q, want %q", lock.APIVersion, LockAPIVersion)
	}
	if lock.Kind != LockKind {
		t.Errorf("kind = %q, want %q", lock.Kind, LockKind)
	}
	if lock.Project.Org != "companyx" {
		t.Errorf("project.org = %q, want %q", lock.Project.Org, "companyx")
	}
	if lock.Project.Name != "payments-api" {
		t.Errorf("project.name = %q, want %q", lock.Project.Name, "payments-api")
	}
	if len(lock.Installs) != 1 {
		t.Fatalf("installs count = %d, want 1", len(lock.Installs))
	}

	install := lock.Installs[0]
	if install.Package != "companyx/platform/agent-backend" {
		t.Errorf("package = %q, want %q", install.Package, "companyx/platform/agent-backend")
	}
	if install.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", install.Version, "1.0.0")
	}
	if install.Digest != "sha256:abc123" {
		t.Errorf("digest = %q, want %q", install.Digest, "sha256:abc123")
	}
	if len(install.Artifacts) != 1 {
		t.Fatalf("artifacts count = %d, want 1", len(install.Artifacts))
	}
	if install.Artifacts[0].Source != "AGENTS.md" {
		t.Errorf("artifact source = %q, want %q", install.Artifacts[0].Source, "AGENTS.md")
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing project org",
			content: "apiVersion: trove.io/v1\nkind: TroveLock\nproject:\n  name: test\ninstalls: []\n",
			wantErr: "project.org is required",
		},
		{
			name:    "missing project name",
			content: "apiVersion: trove.io/v1\nkind: TroveLock\nproject:\n  org: test\ninstalls: []\n",
			wantErr: "project.name is required",
		},
		{
			name:    "missing install package",
			content: "apiVersion: trove.io/v1\nkind: TroveLock\nproject:\n  org: test\n  name: test\ninstalls:\n  - version: 1.0.0\n    digest: sha256:abc\n",
			wantErr: "installs[0].package is required",
		},
		{
			name:    "missing install version",
			content: "apiVersion: trove.io/v1\nkind: TroveLock\nproject:\n  org: test\n  name: test\ninstalls:\n  - package: test/pkg\n    digest: sha256:abc\n",
			wantErr: "installs[0].version is required",
		},
		{
			name:    "missing install digest",
			content: "apiVersion: trove.io/v1\nkind: TroveLock\nproject:\n  org: test\n  name: test\ninstalls:\n  - package: test/pkg\n    version: 1.0.0\n",
			wantErr: "installs[0].digest is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.content))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseDuplicatePackage(t *testing.T) {
	content := `apiVersion: trove.io/v1
kind: TroveLock
project:
  org: test
  name: test
installs:
  - package: test/pkg
    version: 1.0.0
    digest: sha256:abc
  - package: test/pkg
    version: 2.0.0
    digest: sha256:def
`
	_, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for duplicate package, got nil")
	}
}

func TestNewLockFile(t *testing.T) {
	lock := New("trove", "0.1.0", ProjectInfo{
		Org:  "companyx",
		Name: "payments-api",
		Repo: "https://git.company.com/payments/payments-api",
	})

	if lock.APIVersion != LockAPIVersion {
		t.Errorf("apiVersion = %q, want %q", lock.APIVersion, LockAPIVersion)
	}
	if lock.Kind != LockKind {
		t.Errorf("kind = %q, want %q", lock.Kind, LockKind)
	}
	if lock.GeneratedBy.Name != "trove" {
		t.Errorf("generatedBy.name = %q, want %q", lock.GeneratedBy.Name, "trove")
	}
	if lock.Project.Org != "companyx" {
		t.Errorf("project.org = %q, want %q", lock.Project.Org, "companyx")
	}
}

func TestAddAndGetInstall(t *testing.T) {
	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})

	entry := InstallEntry{
		Package:           "test/pkg",
		RequestedSelector: "latest",
		Version:           "1.0.0",
		Digest:            "sha256:abc",
		InstalledAt:       time.Now().UTC(),
		Artifacts: []ArtifactPin{
			{Source: "AGENTS.md", Target: "AGENTS.md", Digest: "sha256:def"},
		},
	}

	lock.AddInstall(entry)

	got := lock.GetInstall("test/pkg")
	if got == nil {
		t.Fatal("getInstall returned nil")
	}
	if got.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", got.Version, "1.0.0")
	}
}

func TestAddInstallUpdatesExisting(t *testing.T) {
	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})

	lock.AddInstall(InstallEntry{
		Package: "test/pkg",
		Version: "1.0.0",
		Digest:  "sha256:abc",
	})

	lock.AddInstall(InstallEntry{
		Package: "test/pkg",
		Version: "2.0.0",
		Digest:  "sha256:def",
	})

	if len(lock.Installs) != 1 {
		t.Fatalf("installs count = %d, want 1", len(lock.Installs))
	}
	if lock.Installs[0].Version != "2.0.0" {
		t.Errorf("version = %q, want %q", lock.Installs[0].Version, "2.0.0")
	}
}

func TestRemoveInstall(t *testing.T) {
	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})

	lock.AddInstall(InstallEntry{Package: "test/pkg-a", Version: "1.0.0", Digest: "sha256:a"})
	lock.AddInstall(InstallEntry{Package: "test/pkg-b", Version: "1.0.0", Digest: "sha256:b"})

	lock.RemoveInstall("test/pkg-a")

	if len(lock.Installs) != 1 {
		t.Fatalf("installs count = %d, want 1", len(lock.Installs))
	}
	if lock.Installs[0].Package != "test/pkg-b" {
		t.Errorf("remaining package = %q, want %q", lock.Installs[0].Package, "test/pkg-b")
	}
}

func TestMarshalAndRoundTrip(t *testing.T) {
	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})
	lock.AddInstall(InstallEntry{
		Package:           "test/pkg",
		RequestedSelector: "latest",
		Version:           "1.0.0",
		Digest:            "sha256:abc",
		InstalledAt:       time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC),
		Artifacts: []ArtifactPin{
			{Source: "AGENTS.md", Target: "AGENTS.md", Digest: "sha256:def"},
		},
	})

	content, err := lock.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	parsed, err := Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.Project.Org != "test" {
		t.Errorf("project.org = %q, want %q", parsed.Project.Org, "test")
	}
	if len(parsed.Installs) != 1 {
		t.Fatalf("installs count = %d, want 1", len(parsed.Installs))
	}
	if parsed.Installs[0].Version != "1.0.0" {
		t.Errorf("version = %q, want %q", parsed.Installs[0].Version, "1.0.0")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultLockFile)

	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})
	lock.AddInstall(InstallEntry{
		Package: "test/pkg",
		Version: "1.0.0",
		Digest:  "sha256:abc",
	})

	if err := lock.WriteFile(path); err != nil {
		t.Fatalf("writeFile: %v", err)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}

	if parsed.Project.Org != "test" {
		t.Errorf("project.org = %q, want %q", parsed.Project.Org, "test")
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/.trove.lock.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseDefaultsAPIVersionAndKind(t *testing.T) {
	content := `project:
  org: test
  name: test
installs: []
`
	lock, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if lock.APIVersion != LockAPIVersion {
		t.Errorf("apiVersion = %q, want %q", lock.APIVersion, LockAPIVersion)
	}
	if lock.Kind != LockKind {
		t.Errorf("kind = %q, want %q", lock.Kind, LockKind)
	}
}

func TestValidateUnsupportedAPIVersion(t *testing.T) {
	content := `apiVersion: trove.io/v2
kind: TroveLock
project:
  org: test
  name: test
installs: []
`
	_, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for unsupported apiVersion, got nil")
	}
}

func TestValidateUnsupportedKind(t *testing.T) {
	content := `apiVersion: trove.io/v1
kind: UnknownKind
project:
  org: test
  name: test
installs: []
`
	_, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for unsupported kind, got nil")
	}
}

func TestWriteFileCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", DefaultLockFile)

	lock := New("trove", "0.1.0", ProjectInfo{Org: "test", Name: "test"})

	err := lock.WriteFile(path)
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := lock.WriteFile(path); err != nil {
		t.Fatalf("writeFile after mkdir: %v", err)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
}
