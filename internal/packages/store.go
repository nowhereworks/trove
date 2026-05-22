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
	GetPackage(ctx context.Context, org string, namespace string, name string) (PackageDetail, error)
}

type WriteStore interface {
	CreateDraftVersion(ctx context.Context, req CreateDraftVersionRequest) (VersionResource, error)
	UploadArtifact(ctx context.Context, req UploadArtifactRequest) (ArtifactResource, error)
	UploadArtifacts(ctx context.Context, req UploadArtifactsRequest) ([]ArtifactResource, error)
	PublishVersion(ctx context.Context, req PublishVersionRequest) (VersionResource, error)
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

func FillResolvedURLs(result ResolvedVersion) ResolvedVersion {
	result.ManifestURL = "/api/v1/packages/" + result.Org + "/" + result.Namespace + "/" + result.Package + "/versions/" + result.ResolvedVersion + "/manifest"
	result.ArchiveURL = "/api/v1/packages/" + result.Org + "/" + result.Namespace + "/" + result.Package + "/versions/" + result.ResolvedVersion + "/archive.tar.gz"
	return result
}
