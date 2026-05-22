package packages

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

type MemoryStore struct {
	packages []memoryPackage
}

type memoryPackage struct {
	summary  PackageSummary
	versions []memoryVersion
}

type memoryVersion struct {
	summary   PackageVersionSummary
	manifest  json.RawMessage
	artifacts []RawArtifact
}

func NewSeedMemoryStore() *MemoryStore {
	content := []byte("# Backend Agent Defaults\n\nUse Go, chi, pgx, and small reviewable changes.\n")
	manifest := json.RawMessage(`{"apiVersion":"trove.io/v1","kind":"AgentArtifactPackage","metadata":{"org":"companyx","namespace":"platform","name":"agent-backend","displayName":"Backend Agent Defaults","description":"Default agent instructions, skills, and commands for backend services.","labels":{"language":"golang","framework":"chi","maturity":"production"},"annotations":{"owner":"platform-engineering"}},"spec":{"version":"1.0.0","channel":"stable","license":"internal","visibility":"public","lifecycle":"published","compatibility":{"tools":[{"name":"opencode","version":">=0.6.0 <2.0.0"}],"models":[{"family":"gpt","minContextWindow":128000}],"runtimes":["linux"]},"artifacts":[{"path":"AGENTS.md","type":"agent-instructions","required":true,"targetPath":"AGENTS.md"}],"dependencies":[],"updatePolicy":{"recommendedChannel":"stable","breakingChangeRequiresManualApproval":true},"maintainers":[{"team":"platform-engineering"}],"links":{"docs":"https://docs.company.com/agent-backend"}}}`)

	return &MemoryStore{packages: []memoryPackage{
		{
			summary: PackageSummary{
				Org:           "companyx",
				Namespace:     "platform",
				Name:          "agent-backend",
				DisplayName:   "Backend Agent Defaults",
				Description:   "Default agent instructions, skills, and commands for backend services.",
				Visibility:    "public",
				Lifecycle:     "active",
				LatestVersion: "1.0.0",
				StableVersion: "1.0.0",
			},
			versions: []memoryVersion{
				{
					summary: PackageVersionSummary{
						Version:     "1.0.0",
						Digest:      "sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050",
						Lifecycle:   "published",
						Channel:     "stable",
						PublishedAt: "2026-05-22T00:00:00Z",
					},
					manifest: manifest,
					artifacts: []RawArtifact{
						{
							Path:         "AGENTS.md",
							ContentType:  "text/markdown; charset=utf-8",
							BlobDigest:   "sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050",
							SizeBytes:    int64(len(content)),
							Content:      content,
							CacheControl: "public, max-age=31536000, immutable",
						},
					},
				},
			},
		},
	}}
}

func (s *MemoryStore) Resolve(ctx context.Context, org string, namespace string, name string, selector string) (ResolvedVersion, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return ResolvedVersion{}, ErrPackageNotFound
	}
	version, err := pkg.resolve(selector)
	if err != nil {
		return ResolvedVersion{}, err
	}

	return FillResolvedURLs(ResolvedVersion{
		Org:             pkg.summary.Org,
		Namespace:       pkg.summary.Namespace,
		Package:         pkg.summary.Name,
		Selector:        selector,
		ResolvedVersion: version.summary.Version,
		Digest:          version.summary.Digest,
	}), nil
}

func (s *MemoryStore) GetManifest(ctx context.Context, org string, namespace string, name string, version string) (Manifest, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return Manifest{}, ErrPackageNotFound
	}
	resolved, ok := pkg.findVersion(version)
	if !ok {
		return Manifest{}, ErrVersionNotFound
	}
	return Manifest{Org: org, Namespace: namespace, Package: name, Version: version, Digest: resolved.summary.Digest, Manifest: resolved.manifest}, nil
}

func (s *MemoryStore) GetRawArtifact(ctx context.Context, org string, namespace string, name string, version string, path string) (RawArtifact, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return RawArtifact{}, ErrPackageNotFound
	}
	resolved, ok := pkg.findVersion(version)
	if !ok {
		return RawArtifact{}, ErrVersionNotFound
	}
	for _, artifact := range resolved.artifacts {
		if artifact.Path == path {
			return artifact, nil
		}
	}
	return RawArtifact{}, ErrArtifactNotFound
}

func (s *MemoryStore) ListPackages(ctx context.Context, params ListPackagesParams) (ListPackagesResult, error) {
	_ = ctx
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	items := make([]PackageSummary, 0, len(s.packages))
	for _, pkg := range s.packages {
		cursor := pkg.summary.Org + "/" + pkg.summary.Namespace + "/" + pkg.summary.Name
		if params.Cursor != "" && cursor <= params.Cursor {
			continue
		}
		items = append(items, pkg.summary)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Org+"/"+items[i].Namespace+"/"+items[i].Name < items[j].Org+"/"+items[j].Namespace+"/"+items[j].Name
	})

	var nextCursor *string
	if len(items) > limit {
		next := items[limit-1].Org + "/" + items[limit-1].Namespace + "/" + items[limit-1].Name
		nextCursor = &next
		items = items[:limit]
	}

	return ListPackagesResult{Items: items, NextCursor: nextCursor}, nil
}

func (s *MemoryStore) GetPackage(ctx context.Context, org string, namespace string, name string) (PackageDetail, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return PackageDetail{}, ErrPackageNotFound
	}
	versions := make([]PackageVersionSummary, 0, len(pkg.versions))
	for _, version := range pkg.versions {
		versions = append(versions, version.summary)
	}
	return PackageDetail{PackageSummary: pkg.summary, Versions: versions}, nil
}

func (s *MemoryStore) findPackage(org string, namespace string, name string) (memoryPackage, bool) {
	for _, pkg := range s.packages {
		if pkg.summary.Org == org && pkg.summary.Namespace == namespace && pkg.summary.Name == name {
			return pkg, true
		}
	}
	return memoryPackage{}, false
}

func (pkg memoryPackage) findVersion(version string) (memoryVersion, bool) {
	version = strings.TrimPrefix(version, "v")
	for _, candidate := range pkg.versions {
		if candidate.summary.Version == version {
			return candidate, true
		}
	}
	return memoryVersion{}, false
}

func (pkg memoryPackage) resolve(selector string) (memoryVersion, error) {
	parsed, err := ParseSelector(selector)
	if err != nil {
		return memoryVersion{}, err
	}

	switch parsed.Kind {
	case SelectorExact:
		if version, ok := pkg.findVersion(parsed.Version); ok {
			return version, nil
		}
	case SelectorChannel:
		for _, version := range pkg.versions {
			if version.summary.Channel == parsed.Value || (parsed.Value == "latest" && version.summary.Version == pkg.summary.LatestVersion) || (parsed.Value == "stable" && version.summary.Version == pkg.summary.StableVersion) {
				return version, nil
			}
		}
	case SelectorDigest:
		for _, version := range pkg.versions {
			if version.summary.Digest == parsed.Value {
				return version, nil
			}
		}
	case SelectorMajor:
		for _, version := range pkg.versions {
			if strings.HasPrefix(version.summary.Version, strconvItoa(parsed.Major)+".") {
				return version, nil
			}
		}
	case SelectorMinor:
		prefix := strconvItoa(parsed.Major) + "." + strconvItoa(parsed.Minor) + "."
		for _, version := range pkg.versions {
			if strings.HasPrefix(version.summary.Version, prefix) {
				return version, nil
			}
		}
	}
	return memoryVersion{}, ErrVersionNotFound
}

func strconvItoa(value int) string {
	return strconv.FormatInt(int64(value), 10)
}
