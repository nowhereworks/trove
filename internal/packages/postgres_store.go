package packages

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"trove/internal/db/sqlc"
)

type PostgresStore struct {
	queries *sqlc.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{queries: sqlc.New(pool)}
}

func (s *PostgresStore) Resolve(ctx context.Context, org string, namespace string, name string, selector string) (ResolvedVersion, error) {
	parsed, err := ParseSelector(selector)
	if err != nil {
		return ResolvedVersion{}, err
	}

	var result ResolvedVersion
	switch parsed.Kind {
	case SelectorExact:
		row, err := s.queries.ResolveExact(ctx, sqlc.ResolveExactParams{Org: org, Namespace: namespace, PackageName: name, Version: parsed.Version})
		if err != nil {
			return ResolvedVersion{}, mapReadError(err)
		}
		result = resolvedFromParts(row.Org, row.Namespace, row.PackageName, selector, row.ResolvedVersion, row.Digest)
	case SelectorChannel:
		row, err := s.queries.ResolveChannel(ctx, sqlc.ResolveChannelParams{Org: org, Namespace: namespace, PackageName: name, Channel: parsed.Value})
		if err != nil {
			return ResolvedVersion{}, mapReadError(err)
		}
		result = resolvedFromParts(row.Org, row.Namespace, row.PackageName, selector, row.ResolvedVersion, row.Digest)
	case SelectorDigest:
		row, err := s.queries.ResolveDigest(ctx, sqlc.ResolveDigestParams{Org: org, Namespace: namespace, PackageName: name, Digest: parsed.Value})
		if err != nil {
			return ResolvedVersion{}, mapReadError(err)
		}
		result = resolvedFromParts(row.Org, row.Namespace, row.PackageName, selector, row.ResolvedVersion, row.Digest)
	case SelectorMajor:
		row, err := s.queries.ResolveMajor(ctx, sqlc.ResolveMajorParams{Org: org, Namespace: namespace, PackageName: name, Major: int4(parsed.Major)})
		if err != nil {
			return ResolvedVersion{}, mapReadError(err)
		}
		result = resolvedFromParts(row.Org, row.Namespace, row.PackageName, selector, row.ResolvedVersion, row.Digest)
	case SelectorMinor:
		row, err := s.queries.ResolveMinor(ctx, sqlc.ResolveMinorParams{Org: org, Namespace: namespace, PackageName: name, Major: int4(parsed.Major), Minor: int4(parsed.Minor)})
		if err != nil {
			return ResolvedVersion{}, mapReadError(err)
		}
		result = resolvedFromParts(row.Org, row.Namespace, row.PackageName, selector, row.ResolvedVersion, row.Digest)
	default:
		return ResolvedVersion{}, ErrInvalidSelector
	}

	return FillResolvedURLs(result), nil
}

func (s *PostgresStore) GetManifest(ctx context.Context, org string, namespace string, name string, version string) (Manifest, error) {
	row, err := s.queries.GetManifest(ctx, sqlc.GetManifestParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: name,
		Version:     strings.TrimPrefix(version, "v"),
	})
	if err != nil {
		return Manifest{}, mapReadError(err)
	}

	return Manifest{
		Org:       row.Org,
		Namespace: row.Namespace,
		Package:   row.PackageName,
		Version:   row.Version,
		Digest:    row.Digest,
		Manifest:  json.RawMessage(row.ManifestJson),
	}, nil
}

func (s *PostgresStore) GetRawArtifact(ctx context.Context, org string, namespace string, name string, version string, path string) (RawArtifact, error) {
	row, err := s.queries.GetRawArtifact(ctx, sqlc.GetRawArtifactParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: name,
		Version:     strings.TrimPrefix(version, "v"),
		Path:        path,
	})
	if err != nil {
		return RawArtifact{}, mapReadError(err)
	}

	return RawArtifact{
		Path:         row.Path,
		ContentType:  row.ContentType,
		BlobDigest:   row.BlobDigest,
		SizeBytes:    row.SizeBytes,
		Content:      row.Content,
		CacheControl: row.CacheControl,
	}, nil
}

func (s *PostgresStore) ListPackages(ctx context.Context, params ListPackagesParams) (ListPackagesResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.queries.ListPackages(ctx, sqlc.ListPackagesParams{Cursor: params.Cursor, PageLimit: int32(limit + 1)})
	if err != nil {
		return ListPackagesResult{}, err
	}

	items := make([]PackageSummary, 0, min(len(rows), limit))
	for _, row := range rows {
		items = append(items, packageSummaryFromRow(row.Org, row.Namespace, row.PackageName, row.DisplayName, row.Description, row.Visibility, row.Lifecycle, row.LatestVersion, row.StableVersion))
	}

	var nextCursor *string
	if len(items) > limit {
		next := items[limit-1].Org + "/" + items[limit-1].Namespace + "/" + items[limit-1].Name
		nextCursor = &next
		items = items[:limit]
	}

	return ListPackagesResult{Items: items, NextCursor: nextCursor}, nil
}

func (s *PostgresStore) GetPackage(ctx context.Context, org string, namespace string, name string) (PackageDetail, error) {
	row, err := s.queries.GetPackageSummary(ctx, sqlc.GetPackageSummaryParams{Org: org, Namespace: namespace, PackageName: name})
	if err != nil {
		return PackageDetail{}, mapReadError(err)
	}

	versionRows, err := s.queries.ListPackageVersions(ctx, sqlc.ListPackageVersionsParams{Org: org, Namespace: namespace, PackageName: name})
	if err != nil {
		return PackageDetail{}, err
	}

	versions := make([]PackageVersionSummary, 0, len(versionRows))
	for _, version := range versionRows {
		versions = append(versions, PackageVersionSummary{
			Version:     version.Version,
			Digest:      version.Digest,
			Lifecycle:   version.Lifecycle,
			Channel:     version.Channel,
			PublishedAt: version.PublishedAt,
		})
	}

	return PackageDetail{
		PackageSummary: packageSummaryFromRow(row.Org, row.Namespace, row.PackageName, row.DisplayName, row.Description, row.Visibility, row.Lifecycle, row.LatestVersion, row.StableVersion),
		Versions:       versions,
	}, nil
}

func resolvedFromParts(org string, namespace string, packageName string, selector string, version string, digest string) ResolvedVersion {
	return ResolvedVersion{
		Org:             org,
		Namespace:       namespace,
		Package:         packageName,
		Selector:        selector,
		ResolvedVersion: version,
		Digest:          digest,
	}
}

func packageSummaryFromRow(org string, namespace string, name string, displayName string, description string, visibility string, lifecycle string, latestVersion string, stableVersion string) PackageSummary {
	return PackageSummary{
		Org:           org,
		Namespace:     namespace,
		Name:          name,
		DisplayName:   displayName,
		Description:   description,
		Visibility:    visibility,
		Lifecycle:     lifecycle,
		LatestVersion: latestVersion,
		StableVersion: stableVersion,
	}
}

func int4(value int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(value), Valid: true}
}

func mapReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPackageNotFound
	}
	return err
}
