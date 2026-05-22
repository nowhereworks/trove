package packages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"trove/internal/manifest"
)

type DigestArtifact struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	TargetPath string `json:"targetPath"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"sizeBytes"`
}

func FileDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func PackageDigest(m manifest.Manifest, artifacts []DigestArtifact) (string, error) {
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})

	payload := struct {
		Manifest  manifest.Manifest `json:"manifest"`
		Artifacts []DigestArtifact  `json:"artifacts"`
	}{Manifest: m, Artifacts: artifacts}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode package digest payload: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
