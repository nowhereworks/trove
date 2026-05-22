package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"trove/internal/manifest"
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

func (s *MemoryStore) CreateDraftVersion(ctx context.Context, req CreateDraftVersionRequest) (VersionResource, error) {
	_ = ctx
	if _, _, _, err := ParseStrictSemver(req.Version); err != nil {
		return VersionResource{}, err
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}

	for packageIndex := range s.packages {
		pkg := &s.packages[packageIndex]
		if pkg.summary.Org != req.Org || pkg.summary.Namespace != req.Namespace || pkg.summary.Name != req.Package {
			continue
		}
		if _, ok := pkg.findVersion(req.Version); ok {
			return VersionResource{}, ErrVersionExists
		}
		now := time.Now().UTC()
		pkg.versions = append(pkg.versions, memoryVersion{summary: PackageVersionSummary{Version: req.Version, Lifecycle: "draft"}, manifest: json.RawMessage(`{}`)})
		return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: req.Version, Lifecycle: "draft", Visibility: req.Visibility, CreatedAt: FormatTime(now), UpdatedAt: FormatTime(now)}, nil
	}
	return VersionResource{}, ErrPackageNotFound
}

func (s *MemoryStore) UploadArtifact(ctx context.Context, req UploadArtifactRequest) (ArtifactResource, error) {
	_ = ctx
	pkgIndex, versionIndex, err := s.findMutableVersion(req.Org, req.Namespace, req.Package, req.Version)
	if err != nil {
		return ArtifactResource{}, err
	}
	version := &s.packages[pkgIndex].versions[versionIndex]

	digest := FileDigest(req.Content)
	artifactType := "artifact"
	targetPath := req.Path
	if req.Path == "trove.yaml" {
		parsed, err := manifest.Parse(req.Content)
		if err != nil {
			return ArtifactResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: req.Version}); err != nil {
			return ArtifactResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		manifestJSON, err := json.Marshal(parsed)
		if err != nil {
			return ArtifactResource{}, err
		}
		version.manifest = manifestJSON
		artifactType = "manifest"
	} else if parsed, ok := decodeStoredManifest(version.manifest); ok {
		for _, artifact := range parsed.Spec.Artifacts {
			if artifact.Path == req.Path {
				artifactType = artifact.Type
				if artifact.TargetPath != "" {
					targetPath = artifact.TargetPath
				}
				break
			}
		}
	}

	resource := RawArtifact{Path: req.Path, ContentType: contentTypeOrDefault(req.ContentType), BlobDigest: digest, SizeBytes: int64(len(req.Content)), Content: req.Content, CacheControl: "private, max-age=31536000, immutable"}
	updated := false
	for i := range version.artifacts {
		if version.artifacts[i].Path == req.Path {
			version.artifacts[i] = resource
			updated = true
			break
		}
	}
	if !updated {
		version.artifacts = append(version.artifacts, resource)
	}
	return ArtifactResource{Path: req.Path, Type: artifactType, ContentType: resource.ContentType, Digest: digest, SizeBytes: resource.SizeBytes, TargetPath: targetPath}, nil
}

func (s *MemoryStore) UploadArtifacts(ctx context.Context, req UploadArtifactsRequest) ([]ArtifactResource, error) {
	ordered := orderManifestFirst(req.Artifacts)
	resources := make([]ArtifactResource, 0, len(ordered))
	for _, artifact := range ordered {
		resource, err := s.UploadArtifact(ctx, UploadArtifactRequest{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: req.Version, Path: artifact.Path, ContentType: artifact.ContentType, Content: artifact.Content})
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func (s *MemoryStore) PublishVersion(ctx context.Context, req PublishVersionRequest) (VersionResource, error) {
	_ = ctx
	pkgIndex, versionIndex, err := s.findMutableVersion(req.Org, req.Namespace, req.Package, req.Version)
	if err != nil {
		return VersionResource{}, err
	}
	pkg := &s.packages[pkgIndex]
	version := &pkg.versions[versionIndex]

	parsed, ok := decodeStoredManifest(version.manifest)
	if !ok || parsed.APIVersion == "" {
		return VersionResource{}, ErrInvalidManifest
	}
	if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: req.Version}); err != nil {
		return VersionResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}

	byPath := map[string]RawArtifact{}
	for _, artifact := range version.artifacts {
		byPath[artifact.Path] = artifact
	}
	digestArtifacts := make([]DigestArtifact, 0, len(parsed.Spec.Artifacts))
	for _, artifact := range parsed.Spec.Artifacts {
		raw, ok := byPath[artifact.Path]
		if !ok {
			return VersionResource{}, fmt.Errorf("%w: %s", ErrMissingArtifact, artifact.Path)
		}
		targetPath := artifact.TargetPath
		if targetPath == "" {
			targetPath = artifact.Path
		}
		digestArtifacts = append(digestArtifacts, DigestArtifact{Path: artifact.Path, Type: artifact.Type, TargetPath: targetPath, Digest: raw.BlobDigest, SizeBytes: raw.SizeBytes})
	}

	digest, err := PackageDigest(parsed, digestArtifacts)
	if err != nil {
		return VersionResource{}, err
	}
	version.summary.Lifecycle = "published"
	version.summary.Digest = digest
	version.summary.Channel = parsed.Spec.Channel
	version.summary.PublishedAt = FormatTime(time.Now().UTC())
	pkg.summary.LatestVersion = version.summary.Version
	if parsed.Spec.Channel == "stable" {
		pkg.summary.StableVersion = version.summary.Version
	}

	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: version.summary.Version, Lifecycle: "published", Visibility: parsed.Spec.Visibility, Digest: digest, PublishedAt: version.summary.PublishedAt}, nil
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

func (s *MemoryStore) findMutableVersion(org string, namespace string, name string, version string) (int, int, error) {
	version = strings.TrimPrefix(version, "v")
	for packageIndex := range s.packages {
		pkg := &s.packages[packageIndex]
		if pkg.summary.Org != org || pkg.summary.Namespace != namespace || pkg.summary.Name != name {
			continue
		}
		for versionIndex := range pkg.versions {
			candidate := &pkg.versions[versionIndex]
			if candidate.summary.Version != version {
				continue
			}
			if candidate.summary.Lifecycle == "published" {
				return 0, 0, ErrVersionImmutable
			}
			return packageIndex, versionIndex, nil
		}
		return 0, 0, ErrVersionNotFound
	}
	return 0, 0, ErrPackageNotFound
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
		targetVersion := ""
		if parsed.Value == "latest" {
			targetVersion = pkg.summary.LatestVersion
		}
		if parsed.Value == "stable" {
			targetVersion = pkg.summary.StableVersion
		}
		if targetVersion != "" {
			if version, ok := pkg.findVersion(targetVersion); ok {
				return version, nil
			}
		}
		for _, version := range pkg.versions {
			if version.summary.Channel == parsed.Value {
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
