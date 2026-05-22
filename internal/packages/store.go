package packages

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrPackageNotFound  = errors.New("package not found")
	ErrVersionNotFound  = errors.New("version not found")
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrInvalidSelector  = errors.New("invalid selector")
)

type Store interface {
	Resolve(ctx context.Context, org string, namespace string, name string, selector string) (ResolvedVersion, error)
	GetManifest(ctx context.Context, org string, namespace string, name string, version string) (Manifest, error)
	GetRawArtifact(ctx context.Context, org string, namespace string, name string, version string, path string) (RawArtifact, error)
	ListPackages(ctx context.Context, params ListPackagesParams) (ListPackagesResult, error)
	GetPackage(ctx context.Context, org string, namespace string, name string) (PackageDetail, error)
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
