package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"trove/internal/db/sqlc"
	"trove/internal/manifest"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: sqlc.New(pool)}
}

func (s *PostgresStore) CreateDraftVersion(ctx context.Context, req CreateDraftVersionRequest) (VersionResource, error) {
	major, minor, patch, err := ParseStrictSemver(req.Version)
	if err != nil {
		return VersionResource{}, err
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}

	packageID, err := s.queries.GetPackageID(ctx, sqlc.GetPackageIDParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package})
	if err != nil {
		return VersionResource{}, mapReadError(err)
	}
	id, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}

	row, err := s.queries.CreateDraftVersion(ctx, sqlc.CreateDraftVersionParams{
		ID:          id,
		PackageID:   packageID,
		Version:     req.Version,
		SemverMajor: int4(major),
		SemverMinor: int4(minor),
		SemverPatch: int4(patch),
		Visibility:  req.Visibility,
	})
	if err != nil {
		return VersionResource{}, mapWriteError(err)
	}

	return VersionResource{
		Org:        req.Org,
		Namespace:  req.Namespace,
		Package:    req.Package,
		Version:    row.Version,
		Lifecycle:  row.Lifecycle,
		Visibility: row.Visibility,
		CreatedAt:  timestamptzString(row.CreatedAt),
		UpdatedAt:  timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) UploadArtifact(ctx context.Context, req UploadArtifactRequest) (ArtifactResource, error) {
	if req.Path == "" || strings.HasPrefix(req.Path, "/") || strings.Contains(req.Path, "..") {
		return ArtifactResource{}, ErrArtifactNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArtifactResource{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	version, err := q.GetVersionForUpdate(ctx, sqlc.GetVersionForUpdateParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package, Version: strings.TrimPrefix(req.Version, "v")})
	if err != nil {
		return ArtifactResource{}, mapReadError(err)
	}
	if version.Lifecycle == "published" {
		return ArtifactResource{}, ErrVersionImmutable
	}

	digest := FileDigest(req.Content)
	if err := q.UpsertArtifactBlob(ctx, sqlc.UpsertArtifactBlobParams{Digest: digest, Content: req.Content, SizeBytes: int64(len(req.Content))}); err != nil {
		return ArtifactResource{}, err
	}
	if err := q.UpsertArtifactLocation(ctx, digest); err != nil {
		return ArtifactResource{}, err
	}

	artifactType := "artifact"
	targetPath := req.Path
	if req.Path == "trove.yaml" {
		parsed, err := manifest.Parse(req.Content)
		if err != nil {
			return ArtifactResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: version.Version}); err != nil {
			return ArtifactResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		manifestJSON, err := json.Marshal(parsed)
		if err != nil {
			return ArtifactResource{}, err
		}
		if err := q.UpdateVersionManifest(ctx, sqlc.UpdateVersionManifestParams{ManifestJson: manifestJSON, Visibility: parsed.Spec.Visibility, Channel: textOrNull(parsed.Spec.Channel), ID: version.ID}); err != nil {
			return ArtifactResource{}, err
		}
		artifactType = "manifest"
	} else if parsed, ok := decodeStoredManifest(version.ManifestJson); ok {
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

	id, err := newUUID()
	if err != nil {
		return ArtifactResource{}, err
	}
	row, err := q.UpsertArtifactFile(ctx, sqlc.UpsertArtifactFileParams{
		ID:               id,
		PackageVersionID: version.ID,
		Path:             req.Path,
		Type:             artifactType,
		ContentType:      contentTypeOrDefault(req.ContentType),
		BlobDigest:       digest,
		SizeBytes:        int64(len(req.Content)),
		TargetPath:       textOrNull(targetPath),
	})
	if err != nil {
		return ArtifactResource{}, mapWriteError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ArtifactResource{}, err
	}

	return ArtifactResource{Path: row.Path, Type: row.Type, ContentType: row.ContentType, Digest: row.BlobDigest, SizeBytes: row.SizeBytes, TargetPath: textValue(row.TargetPath)}, nil
}

func (s *PostgresStore) PublishVersion(ctx context.Context, req PublishVersionRequest) (VersionResource, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return VersionResource{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	version, err := q.GetVersionForUpdate(ctx, sqlc.GetVersionForUpdateParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package, Version: strings.TrimPrefix(req.Version, "v")})
	if err != nil {
		return VersionResource{}, mapReadError(err)
	}
	if version.Lifecycle == "published" {
		return VersionResource{}, ErrVersionImmutable
	}

	parsed, ok := decodeStoredManifest(version.ManifestJson)
	if !ok || parsed.APIVersion == "" {
		return VersionResource{}, ErrInvalidManifest
	}
	if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: version.Version}); err != nil {
		return VersionResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}

	rows, err := q.ListArtifactMetadataForVersion(ctx, version.ID)
	if err != nil {
		return VersionResource{}, err
	}
	byPath := map[string]sqlc.ListArtifactMetadataForVersionRow{}
	for _, row := range rows {
		byPath[row.Path] = row
	}

	digestArtifacts := make([]DigestArtifact, 0, len(parsed.Spec.Artifacts))
	for _, artifact := range parsed.Spec.Artifacts {
		row, ok := byPath[artifact.Path]
		if !ok {
			return VersionResource{}, fmt.Errorf("%w: %s", ErrMissingArtifact, artifact.Path)
		}
		targetPath := artifact.TargetPath
		if targetPath == "" {
			targetPath = artifact.Path
		}
		digestArtifacts = append(digestArtifacts, DigestArtifact{Path: artifact.Path, Type: artifact.Type, TargetPath: targetPath, Digest: row.BlobDigest, SizeBytes: row.SizeBytes})
	}

	digest, err := PackageDigest(parsed, digestArtifacts)
	if err != nil {
		return VersionResource{}, err
	}
	published, err := q.PublishVersion(ctx, sqlc.PublishVersionParams{Digest: textOrNull(digest), ID: version.ID})
	if err != nil {
		return VersionResource{}, mapWriteError(err)
	}

	latestID, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}
	if err := q.UpsertChannel(ctx, sqlc.UpsertChannelParams{ID: latestID, PackageID: version.PackageID, Name: "latest", PackageVersionID: version.ID}); err != nil {
		return VersionResource{}, err
	}
	if parsed.Spec.Channel == "stable" {
		stableID, err := newUUID()
		if err != nil {
			return VersionResource{}, err
		}
		if err := q.UpsertChannel(ctx, sqlc.UpsertChannelParams{ID: stableID, PackageID: version.PackageID, Name: "stable", PackageVersionID: version.ID}); err != nil {
			return VersionResource{}, err
		}
	}

	auditID, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}
	if err := q.InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{ID: auditID, PackageVersionID: version.ID, ActorServiceAccount: textOrNull("dev"), Action: "package.published", MetadataJson: []byte(`{"source":"api"}`), PackageID: version.PackageID}); err != nil {
		return VersionResource{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return VersionResource{}, err
	}

	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: published.Version, Lifecycle: published.Lifecycle, Visibility: parsed.Spec.Visibility, Digest: textValue(published.Digest), PublishedAt: timestamptzString(published.PublishedAt)}, nil
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

func textOrNull(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timestamptzString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return FormatTime(value.Time)
}

func contentTypeOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "application/octet-stream"
	}
	return value
}

func decodeStoredManifest(data []byte) (manifest.Manifest, bool) {
	if len(data) == 0 || string(data) == "{}" {
		return manifest.Manifest{}, false
	}
	var parsed manifest.Manifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		return manifest.Manifest{}, false
	}
	return parsed, true
}

func mapReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPackageNotFound
	}
	return err
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrVersionExists
	}
	return err
}
