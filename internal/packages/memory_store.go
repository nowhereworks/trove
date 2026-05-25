package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"trove/internal/archives"
	"trove/internal/manifest"
)

type MemoryStore struct {
	packages   []memoryPackage
	orgs       []OrgResource
	namespaces []NamespaceResource
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

	return &MemoryStore{orgs: []OrgResource{
		{
			ID:          "org_companyx",
			Slug:        "companyx",
			DisplayName: "Company X",
			Visibility:  "public",
			CreatedAt:   "2026-05-22T00:00:00Z",
			UpdatedAt:   "2026-05-22T00:00:00Z",
		},
	}, namespaces: []NamespaceResource{
		{
			ID:          "ns_companyx_platform",
			OrgID:       "org_companyx",
			Slug:        "platform",
			DisplayName: "Platform",
			Visibility:  "public",
			CreatedAt:   "2026-05-22T00:00:00Z",
			UpdatedAt:   "2026-05-22T00:00:00Z",
		},
	}, packages: []memoryPackage{
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

func (s *MemoryStore) DeprecateVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
	_ = ctx
	pkgIndex, versionIndex, err := s.findPublishedVersion(req.Org, req.Namespace, req.Package, req.Version)
	if err != nil {
		return VersionResource{}, err
	}
	version := &s.packages[pkgIndex].versions[versionIndex]
	if version.summary.Lifecycle != "published" {
		return VersionResource{}, fmt.Errorf("version %s is not published", version.summary.Version)
	}
	version.summary.Lifecycle = "deprecated"
	now := FormatTime(time.Now().UTC())
	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: version.summary.Version, Lifecycle: "deprecated", UpdatedAt: now}, nil
}

func (s *MemoryStore) YankVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
	_ = ctx
	pkgIndex, versionIndex, err := s.findPublishedVersion(req.Org, req.Namespace, req.Package, req.Version)
	if err != nil {
		return VersionResource{}, err
	}
	version := &s.packages[pkgIndex].versions[versionIndex]
	if version.summary.Lifecycle != "published" && version.summary.Lifecycle != "deprecated" {
		return VersionResource{}, fmt.Errorf("version %s cannot be yanked", version.summary.Version)
	}
	version.summary.Lifecycle = "yanked"
	now := FormatTime(time.Now().UTC())
	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: version.summary.Version, Lifecycle: "yanked", UpdatedAt: now}, nil
}

func (s *MemoryStore) CreateOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error) {
	if _, ok := s.findOrg(req.Slug); ok {
		return OrgResource{}, ErrVersionExists
	}
	return s.appendOrg(ctx, req), nil
}

func (s *MemoryStore) EnsureOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error) {
	_ = ctx
	if org, ok := s.findOrg(req.Slug); ok {
		return org, nil
	}
	return s.appendOrg(ctx, req), nil
}

func (s *MemoryStore) CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error) {
	if _, ok := s.findNamespace(req.Org, req.Slug); ok {
		return NamespaceResource{}, ErrVersionExists
	}
	return s.appendNamespace(ctx, req)

}

func (s *MemoryStore) EnsureNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error) {
	if namespace, ok := s.findNamespace(req.Org, req.Slug); ok {
		return namespace, nil
	}
	return s.appendNamespace(ctx, req)
}

func (s *MemoryStore) CreatePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error) {
	if _, ok := s.findPackage(req.Org, req.Namespace, req.Name); ok {
		return PackageResource{}, ErrVersionExists
	}
	return s.appendPackage(ctx, req)
}

func (s *MemoryStore) EnsurePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error) {
	if pkg, ok := s.findPackage(req.Org, req.Namespace, req.Name); ok {
		return packageResourceFromSummary(pkg.summary), nil
	}
	return s.appendPackage(ctx, req)
}

func (s *MemoryStore) CreateProject(ctx context.Context, req CreateProjectRequest) (ProjectResource, error) {
	_ = ctx
	return ProjectResource{
		ID:        "proj_stub",
		Org:       req.Org,
		Name:      req.Name,
		RepoURL:   req.RepoURL,
		CreatedAt: FormatTime(time.Now().UTC()),
	}, nil
}

func (s *MemoryStore) ReportProjectAdoption(ctx context.Context, req ReportProjectAdoptionRequest) error {
	_ = ctx
	return nil
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

func (s *MemoryStore) GetArchive(ctx context.Context, org string, namespace string, name string, version string, format ArchiveFormat) (Archive, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return Archive{}, ErrPackageNotFound
	}
	resolved, ok := pkg.findVersion(version)
	if !ok {
		return Archive{}, ErrVersionNotFound
	}
	parsed, ok := decodeStoredManifest(resolved.manifest)
	if !ok {
		return Archive{}, ErrInvalidManifest
	}

	byPath := map[string]RawArtifact{}
	for _, artifact := range resolved.artifacts {
		byPath[artifact.Path] = artifact
	}
	files := make([]archives.File, 0, len(parsed.Spec.Artifacts))
	for _, artifact := range parsed.Spec.Artifacts {
		raw, ok := byPath[artifact.Path]
		if !ok {
			return Archive{}, ErrArtifactNotFound
		}
		files = append(files, archives.File{Path: artifact.Path, Content: raw.Content})
	}

	content, contentType, err := generateArchive(format, files)
	if err != nil {
		return Archive{}, err
	}
	cacheControl := "private, max-age=31536000, immutable"
	if pkg.summary.Visibility == "public" {
		cacheControl = "public, max-age=31536000, immutable"
	}
	return Archive{ContentType: contentType, ETag: resolved.summary.Digest, CacheControl: cacheControl, Content: content}, nil
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

func (s *MemoryStore) CheckVisibility(ctx context.Context, org, namespace, name, version string) (string, error) {
	_ = ctx
	pkg, ok := s.findPackage(org, namespace, name)
	if !ok {
		return "", ErrPackageNotFound
	}
	if version == "" || version == "latest" {
		return pkg.summary.Visibility, nil
	}
	ver, ok := pkg.findVersion(version)
	if !ok {
		return pkg.summary.Visibility, nil
	}
	visibility := pkg.summary.Visibility
	if ver.summary.Lifecycle == "published" {
		if parsed, ok := decodeStoredManifest(ver.manifest); ok && parsed.Spec.Visibility != "" {
			visibility = parsed.Spec.Visibility
		}
	}
	return visibility, nil
}

func (s *MemoryStore) SearchPackages(ctx context.Context, params SearchParams) (SearchResult, error) {
	_ = ctx
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := strings.ToLower(params.Query)
	items := make([]PackageSummary, 0, len(s.packages))
	for _, pkg := range s.packages {
		if pkg.summary.Visibility != "public" {
			continue
		}
		if params.Org != "" && pkg.summary.Org != params.Org {
			continue
		}
		if params.Namespace != "" && pkg.summary.Namespace != params.Namespace {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(pkg.summary.Name + " " + pkg.summary.DisplayName + " " + pkg.summary.Description)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
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

	return SearchResult{Items: items, NextCursor: nextCursor}, nil
}

func (s *MemoryStore) GetPackageAdoption(ctx context.Context, org string, namespace string, name string) (PackageAdoption, error) {
	_ = ctx
	_, ok := s.findPackage(org, namespace, name)
	if !ok {
		return PackageAdoption{}, ErrPackageNotFound
	}
	return PackageAdoption{ProjectCount: 0, VersionCount: 0, Versions: nil}, nil
}

func (s *MemoryStore) findPackage(org string, namespace string, name string) (memoryPackage, bool) {
	for _, pkg := range s.packages {
		if pkg.summary.Org == org && pkg.summary.Namespace == namespace && pkg.summary.Name == name {
			return pkg, true
		}
	}
	return memoryPackage{}, false
}

func (s *MemoryStore) findOrg(slug string) (OrgResource, bool) {
	for _, org := range s.orgs {
		if org.Slug == slug {
			return org, true
		}
	}
	return OrgResource{}, false
}

func (s *MemoryStore) findNamespace(org string, slug string) (NamespaceResource, bool) {
	orgResource, ok := s.findOrg(org)
	if !ok {
		return NamespaceResource{}, false
	}
	for _, namespace := range s.namespaces {
		if namespace.OrgID == orgResource.ID && namespace.Slug == slug {
			return namespace, true
		}
	}
	return NamespaceResource{}, false

}

func (s *MemoryStore) appendOrg(ctx context.Context, req CreateOrgRequest) OrgResource {
	_ = ctx
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	now := FormatTime(time.Now().UTC())
	org := OrgResource{ID: "org_" + req.Slug, Slug: req.Slug, DisplayName: req.DisplayName, Visibility: req.Visibility, CreatedAt: now, UpdatedAt: now}
	s.orgs = append(s.orgs, org)
	return org
}

func (s *MemoryStore) appendNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error) {
	_ = ctx
	org, ok := s.findOrg(req.Org)
	if !ok {
		return NamespaceResource{}, ErrPackageNotFound
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	now := FormatTime(time.Now().UTC())
	namespace := NamespaceResource{ID: "ns_" + req.Org + "_" + req.Slug, OrgID: org.ID, Slug: req.Slug, DisplayName: req.DisplayName, Visibility: req.Visibility, CreatedAt: now, UpdatedAt: now}
	s.namespaces = append(s.namespaces, namespace)
	return namespace, nil
}

func (s *MemoryStore) appendPackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error) {
	_ = ctx
	namespace, ok := s.findNamespace(req.Org, req.Namespace)
	if !ok {
		return PackageResource{}, ErrPackageNotFound
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}
	now := FormatTime(time.Now().UTC())
	summary := PackageSummary{Org: req.Org, Namespace: req.Namespace, Name: req.Name, DisplayName: req.DisplayName, Description: req.Description, Visibility: req.Visibility, Lifecycle: "active"}
	s.packages = append(s.packages, memoryPackage{summary: summary})
	resource := packageResourceFromSummary(summary)
	resource.ID = "pkg_" + req.Org + "_" + req.Namespace + "_" + req.Name
	resource.NamespaceID = namespace.ID
	resource.CreatedAt = now
	resource.UpdatedAt = now
	return resource, nil
}

func packageResourceFromSummary(summary PackageSummary) PackageResource {
	return PackageResource{Org: summary.Org, Namespace: summary.Namespace, Name: summary.Name, DisplayName: summary.DisplayName, Description: summary.Description, Visibility: summary.Visibility, Lifecycle: summary.Lifecycle}
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

func (s *MemoryStore) findPublishedVersion(org string, namespace string, name string, version string) (int, int, error) {
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
			if candidate.summary.Lifecycle != "published" && candidate.summary.Lifecycle != "deprecated" {
				return 0, 0, fmt.Errorf("version %s cannot be modified", version)
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
