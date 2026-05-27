package cli

import (
	"os"
	"strings"
	"testing"

	"trove/internal/manifest"
)

func TestRemoteAddPreservesMetadataWhenURLHasNoPackage(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	existing := manifest.Manifest{
		APIVersion: manifest.APIVersion,
		Kind:       manifest.Kind,
		Metadata: manifest.Metadata{
			Org:         "nwks",
			Namespace:   "platform",
			Name:        "my-agent",
			Description: "My agent",
		},
	}
	data, err := yamlMarshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("Trovefile", data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := remoteAdd("origin", "https://trove.example.com", false); err != nil {
		t.Fatalf("remoteAdd = %v", err)
	}

	m, err := loadTrovefile("Trovefile")
	if err != nil {
		t.Fatal(err)
	}
	if m.Metadata.Org != "nwks" || m.Metadata.Namespace != "platform" || m.Metadata.Name != "my-agent" {
		t.Errorf("metadata corrupted: org=%q namespace=%q name=%q", m.Metadata.Org, m.Metadata.Namespace, m.Metadata.Name)
	}
	if m.Local == nil || len(m.Local.Remotes) != 1 {
		t.Fatal("remote not added")
	}
	if m.Local.Remotes["origin"].ServerURL != "https://trove.example.com" {
		t.Errorf("serverUrl = %q", m.Local.Remotes["origin"].ServerURL)
	}
}

func TestRemoteAddSetsMetadataFromFullURL(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := remoteAdd("origin", "https://trove.example.com/nwks/platform/my-agent", false); err != nil {
		t.Fatalf("remoteAdd = %v", err)
	}

	m, err := loadTrovefile("Trovefile")
	if err != nil {
		t.Fatal(err)
	}
	if m.Metadata.Org != "nwks" || m.Metadata.Namespace != "platform" || m.Metadata.Name != "my-agent" {
		t.Errorf("metadata not set: org=%q namespace=%q name=%q", m.Metadata.Org, m.Metadata.Namespace, m.Metadata.Name)
	}
}

func TestRemoteAddPreservesNonDefaultOrg(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	existing := manifest.Manifest{
		APIVersion: manifest.APIVersion,
		Kind:       manifest.Kind,
		Metadata: manifest.Metadata{
			Org:         "myorg",
			Namespace:   "myns",
			Name:        "mypkg",
			Description: "My package",
		},
	}
	data, err := yamlMarshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("Trovefile", data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := remoteAdd("origin", "https://trove.example.com/other/ns/pkg", false); err != nil {
		t.Fatalf("remoteAdd = %v", err)
	}

	m, err := loadTrovefile("Trovefile")
	if err != nil {
		t.Fatal(err)
	}
	if m.Metadata.Org != "myorg" || m.Metadata.Namespace != "myns" || m.Metadata.Name != "mypkg" {
		t.Errorf("metadata overwritten: org=%q namespace=%q name=%q", m.Metadata.Org, m.Metadata.Namespace, m.Metadata.Name)
	}
}

func TestRemoteAddServerURLOnlyNoPackage(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	stdout := captureStdout(t, func() error {
		return remoteAdd("origin", "https://trove.example.com", false)
	})
	if !strings.Contains(stdout, "Added remote origin -> https://trove.example.com/") {
		t.Fatalf("stdout = %q", stdout)
	}

	m, err := loadTrovefile("Trovefile")
	if err != nil {
		t.Fatal(err)
	}
	if m.Local.Remotes["origin"].Package != "" {
		t.Errorf("package should be empty, got %q", m.Local.Remotes["origin"].Package)
	}
}
