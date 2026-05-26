package packages

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrPackageNotFound  = errors.New("package not found")
	ErrVersionNotFound  = errors.New("version not found")
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrInvalidSelector  = errors.New("invalid selector")
	ErrVersionExists    = errors.New("version already exists")
	ErrVersionImmutable = errors.New("version immutable")
	ErrInvalidManifest  = errors.New("invalid manifest")
	ErrMissingArtifact  = errors.New("required artifact missing")
	ErrInvalidArchive   = errors.New("invalid archive format")
)

type Store interface {
	Resolve(ctx context.Context, org string, namespace string, name string, selector string) (ResolvedVersion, error)
	GetManifest(ctx context.Context, org string, namespace string, name string, version string) (Manifest, error)
	GetRawArtifact(ctx context.Context, org string, namespace string, name string, version string, path string) (RawArtifact, error)
	GetArchive(ctx context.Context, org string, namespace string, name string, version string, format ArchiveFormat) (Archive, error)
	ListPackages(ctx context.Context, params ListPackagesParams) (ListPackagesResult, error)
	SearchPackages(ctx context.Context, params SearchParams) (SearchResult, error)
	GetPackage(ctx context.Context, org string, namespace string, name string) (PackageDetail, error)
	CheckVisibility(ctx context.Context, org string, namespace string, name string, version string) (string, error)
	GetPackageAdoption(ctx context.Context, org string, namespace string, name string) (PackageAdoption, error)
}

type WriteStore interface {
	CreateDraftVersion(ctx context.Context, req CreateDraftVersionRequest) (VersionResource, error)
	UploadArtifact(ctx context.Context, req UploadArtifactRequest) (ArtifactResource, error)
	UploadArtifacts(ctx context.Context, req UploadArtifactsRequest) ([]ArtifactResource, error)
	PublishVersion(ctx context.Context, req PublishVersionRequest) (VersionResource, error)
	DeprecateVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error)
	YankVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error)
	EnsureOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error)
	CreateOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error)
	CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error)
	EnsureNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error)
	CreatePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error)
	EnsurePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error)
	CreateProject(ctx context.Context, req CreateProjectRequest) (ProjectResource, error)
	ReportProjectAdoption(ctx context.Context, req ReportProjectAdoptionRequest) error
}

type ManagementStore interface {
	GetPackageVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error)
	ResetUnpublishedVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error)
}

type LifecycleChangeRequest struct {
	Org       string
	Namespace string
	Package   string
	Version   string
}

type CreateOrgRequest struct {
	Slug        string
	DisplayName string
	Visibility  string
}

type CreateNamespaceRequest struct {
	Org         string
	Slug        string
	DisplayName string
	Visibility  string
}

type CreatePackageRequest struct {
	Org         string
	Namespace   string
	Name        string
	DisplayName string
	Description string
	Visibility  string
}

type CreateProjectRequest struct {
	Org     string
	Name    string
	RepoURL string
}

type ReportProjectAdoptionRequest struct {
	Org       string
	Name      string
	RepoURL   string
	Installed []InstalledPackage
}

type InstalledPackage struct {
	Package string
	Version string
	Digest  string
}

type ProjectResource struct {
	ID        string `json:"id"`
	Org       string `json:"org"`
	Name      string `json:"name"`
	RepoURL   string `json:"repoUrl"`
	CreatedAt string `json:"createdAt"`
}

type OrgResource struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type NamespaceResource struct {
	ID          string `json:"id"`
	OrgID       string `json:"orgId"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type PackageResource struct {
	ID          string `json:"id"`
	NamespaceID string `json:"namespaceId"`
	Org         string `json:"org"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Lifecycle   string `json:"lifecycle"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CreateDraftVersionRequest struct {
	Org        string
	Namespace  string
	Package    string
	Version    string
	Visibility string
}

type UploadArtifactRequest struct {
	Org         string
	Namespace   string
	Package     string
	Version     string
	Path        string
	ContentType string
	Content     []byte
}

type UploadArtifactsRequest struct {
	Org       string
	Namespace string
	Package   string
	Version   string
	Artifacts []UploadArchiveArtifact
}

type UploadArchiveArtifact struct {
	Path        string
	ContentType string
	Content     []byte
}

type PublishVersionRequest struct {
	Org       string
	Namespace string
	Package   string
	Version   string
}

type VersionResource struct {
	Org         string `json:"org"`
	Namespace   string `json:"namespace"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Lifecycle   string `json:"lifecycle"`
	Visibility  string `json:"visibility"`
	Digest      string `json:"digest,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type ArtifactResource struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	ContentType string `json:"contentType"`
	Digest      string `json:"digest"`
	SizeBytes   int64  `json:"sizeBytes"`
	TargetPath  string `json:"targetPath,omitempty"`
}

type ResolvedVersion struct {
	Org             string `json:"org"`
	Namespace       string `json:"namespace"`
	Package         string `json:"package"`
	Selector        string `json:"selector"`
	ResolvedVersion string `json:"resolvedVersion"`
	Digest          string `json:"digest"`
	ManifestURL     string `json:"manifestUrl"`
	ArchiveURL      string `json:"archiveUrl"`
}

type Manifest struct {
	Org       string          `json:"org"`
	Namespace string          `json:"namespace"`
	Package   string          `json:"package"`
	Version   string          `json:"version"`
	Digest    string          `json:"digest"`
	Manifest  json.RawMessage `json:"manifest"`
}

type RawArtifact struct {
	Path         string
	ContentType  string
	BlobDigest   string
	SizeBytes    int64
	Content      []byte
	CacheControl string
}

type ArchiveFormat string

const (
	ArchiveTarGz ArchiveFormat = "tar.gz"
	ArchiveZip   ArchiveFormat = "zip"
)

type Archive struct {
	ContentType  string
	ETag         string
	CacheControl string
	Content      []byte
}

type PackageSummary struct {
	Org           string `json:"org"`
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"`
	Lifecycle     string `json:"lifecycle"`
	LatestVersion string `json:"latestVersion,omitempty"`
	StableVersion string `json:"stableVersion,omitempty"`
}

type PackageDetail struct {
	PackageSummary
	Versions []PackageVersionSummary `json:"versions"`
}

type PackageVersionSummary struct {
	Version     string `json:"version"`
	Digest      string `json:"digest"`
	Lifecycle   string `json:"lifecycle"`
	Channel     string `json:"channel,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

type ListPackagesParams struct {
	Limit  int
	Cursor string
}

type ListPackagesResult struct {
	Items      []PackageSummary `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type SearchParams struct {
	Query        string
	Org          string
	Namespace    string
	ArtifactType string
	Tool         string
	Limit        int
	Cursor       string
}

type SearchResult struct {
	Items      []PackageSummary `json:"items"`
	NextCursor *string          `json:"nextCursor"`
	TotalRank  float32          `json:"-"`
}

type PackageAdoption struct {
	ProjectCount int64                    `json:"projectCount"`
	VersionCount int64                    `json:"versionCount"`
	Versions     []AdoptionVersionSummary `json:"versions"`
}

type AdoptionVersionSummary struct {
	Version      string `json:"version"`
	InstallCount int64  `json:"installCount"`
}

func FillResolvedURLs(result ResolvedVersion) ResolvedVersion {
	result.ManifestURL = "/api/v1/packages/" + result.Org + "/" + result.Namespace + "/" + result.Package + "/versions/" + result.ResolvedVersion + "/manifest"
	result.ArchiveURL = "/api/v1/packages/" + result.Org + "/" + result.Namespace + "/" + result.Package + "/versions/" + result.ResolvedVersion + "/archive.tar.gz"
	return result
}
