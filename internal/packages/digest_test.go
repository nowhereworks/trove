package packages

import (
	"strings"
	"testing"

	"trove/internal/manifest"
)

func TestFileDigest(t *testing.T) {
	got := FileDigest([]byte("hello"))
	if got != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("FileDigest() = %q", got)
	}
}

func TestPackageDigestSortsArtifacts(t *testing.T) {
	m := manifest.Manifest{APIVersion: manifest.APIVersion, Kind: manifest.Kind}
	a := []DigestArtifact{{Path: "b", Type: "skill", Digest: "sha256:b", SizeBytes: 2}, {Path: "a", Type: "skill", Digest: "sha256:a", SizeBytes: 1}}
	b := []DigestArtifact{{Path: "a", Type: "skill", Digest: "sha256:a", SizeBytes: 1}, {Path: "b", Type: "skill", Digest: "sha256:b", SizeBytes: 2}}

	digestA, err := PackageDigest(m, a)
	if err != nil {
		t.Fatalf("PackageDigest() error = %v", err)
	}
	digestB, err := PackageDigest(m, b)
	if err != nil {
		t.Fatalf("PackageDigest() error = %v", err)
	}
	if digestA != digestB {
		t.Fatalf("digests differ: %q != %q", digestA, digestB)
	}
	if !strings.HasPrefix(digestA, "sha256:") {
		t.Fatalf("digest = %q, want sha256 prefix", digestA)
	}
}
