package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"trove/internal/archives"
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

func (s *PostgresStore) Queries() *sqlc.Queries {
	return s.queries
}

func (s *PostgresStore) CheckVisibility(ctx context.Context, org, namespace, name, version string) (string, error) {
	vis, err := s.queries.GetPackageVisibility(ctx, sqlc.GetPackageVisibilityParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: name,
	})
	if err != nil {
		return "", mapReadError(err)
	}
	return vis, nil
}

func (s *PostgresStore) CreateDraftVersion(ctx context.Context, req CreateDraftVersionRequest) (VersionResource, error) {
	major, minor, patch, err := ParseStrictSemver(req.Version)
	if err != nil {
		return VersionResource{}, err
	}

	packageID, err := s.queries.GetPackageID(ctx, sqlc.GetPackageIDParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package})
	if err != nil {
		return VersionResource{}, mapReadError(err)
	}
	id, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}

	pkgVis, err := s.queries.GetPackageVisibility(ctx, sqlc.GetPackageVisibilityParams{
		Org:         req.Org,
		Namespace:   req.Namespace,
		PackageName: req.Package,
	})
	if err != nil {
		return VersionResource{}, mapReadError(err)
	}

	row, err := s.queries.CreateDraftVersion(ctx, sqlc.CreateDraftVersionParams{
		ID:          id,
		PackageID:   packageID,
		Version:     req.Version,
		SemverMajor: int4(major),
		SemverMinor: int4(minor),
		SemverPatch: int4(patch),
		Visibility:  pkgVis,
	})
	if err != nil {
		return VersionResource{}, mapWriteError(err)
	}

	return VersionResource{
		Org:       req.Org,
		Namespace: req.Namespace,
		Package:   req.Package,
		Version:   row.Version,
		Lifecycle: row.Lifecycle,
		CreatedAt: timestamptzString(row.CreatedAt),
		UpdatedAt: timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) GetPackageVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
	row, err := s.queries.GetPackageVersionAny(ctx, sqlc.GetPackageVersionAnyParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package, Version: strings.TrimPrefix(req.Version, "v")})
	if err != nil {
		return VersionResource{}, mapVersionReadError(err)
	}
	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: row.Version, Lifecycle: row.Lifecycle, Digest: row.Digest, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, PublishedAt: row.PublishedAt}, nil
}

func (s *PostgresStore) ResetUnpublishedVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
	existing, err := s.GetPackageVersion(ctx, req)
	if err != nil {
		return VersionResource{}, err
	}
	if existing.Lifecycle != "draft" && existing.Lifecycle != "review" {
		return VersionResource{}, ErrVersionImmutable
	}
	row, err := s.queries.ResetUnpublishedVersionToDraft(ctx, sqlc.ResetUnpublishedVersionToDraftParams{Org: req.Org, Namespace: req.Namespace, PackageName: req.Package, Version: strings.TrimPrefix(req.Version, "v")})
	if err != nil {
		return VersionResource{}, mapVersionReadError(err)
	}
	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: row.Version, Lifecycle: row.Lifecycle, Digest: row.Digest, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, PublishedAt: row.PublishedAt}, nil
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
		if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package}); err != nil {
			return ArtifactResource{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		manifestJSON, err := json.Marshal(parsed)
		if err != nil {
			return ArtifactResource{}, err
		}
		if err := q.UpdateVersionManifest(ctx, sqlc.UpdateVersionManifestParams{ManifestJson: manifestJSON, ID: version.ID}); err != nil {
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

func (s *PostgresStore) UploadArtifacts(ctx context.Context, req UploadArtifactsRequest) ([]ArtifactResource, error) {
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
	if err := manifest.Validate(parsed, manifest.ValidateOptions{Org: req.Org, Namespace: req.Namespace, Package: req.Package}); err != nil {
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

	auditID, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}
	if err := q.InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{ID: auditID, PackageVersionID: version.ID, ActorServiceAccount: textOrNull("dev"), Action: "package.published", MetadataJson: []byte(`{"source":"api"}`), PackageID: version.PackageID}); err != nil {
		return VersionResource{}, err
	}

	pkgVis, err := s.queries.GetPackageVisibility(ctx, sqlc.GetPackageVisibilityParams{
		Org:         req.Org,
		Namespace:   req.Namespace,
		PackageName: req.Package,
	})
	if err != nil {
		return VersionResource{}, mapReadError(err)
	}

	searchText := buildSearchText(parsed, req.Org, req.Namespace, req.Package, rows)
	labelsJSON, _ := json.Marshal(parsed.Metadata.Labels)
	artifactTypes := extractArtifactTypes(parsed)
	toolNames := extractToolNames(parsed)
	if err := q.UpsertSearchDocument(ctx, sqlc.UpsertSearchDocumentParams{
		PackageID:                version.PackageID,
		LatestPublishedVersionID: version.ID,
		SearchText:               []byte(searchText),
		LabelsJson:               labelsJSON,
		ArtifactTypes:            artifactTypes,
		ToolNames:                toolNames,
		Lifecycle:                "active",
		Visibility:               pkgVis,
	}); err != nil {
		return VersionResource{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return VersionResource{}, err
	}

	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: published.Version, Lifecycle: published.Lifecycle, Digest: textValue(published.Digest), PublishedAt: timestamptzString(published.PublishedAt)}, nil
}

func (s *PostgresStore) DeprecateVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
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
	if version.Lifecycle != "published" {
		return VersionResource{}, fmt.Errorf("version %s is not published", version.Version)
	}

	result, err := q.DeprecateVersion(ctx, version.ID)
	if err != nil {
		return VersionResource{}, mapWriteError(err)
	}

	auditID, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}
	if err := q.InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{ID: auditID, PackageVersionID: version.ID, ActorServiceAccount: textOrNull("dev"), Action: "version.deprecated", MetadataJson: []byte(`{"source":"api"}`), PackageID: version.PackageID}); err != nil {
		return VersionResource{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return VersionResource{}, err
	}

	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: result.Version, Lifecycle: result.Lifecycle, UpdatedAt: timestamptzString(result.UpdatedAt)}, nil
}

func (s *PostgresStore) YankVersion(ctx context.Context, req LifecycleChangeRequest) (VersionResource, error) {
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
	if version.Lifecycle != "published" && version.Lifecycle != "deprecated" {
		return VersionResource{}, fmt.Errorf("version %s cannot be yanked", version.Version)
	}

	result, err := q.YankVersion(ctx, version.ID)
	if err != nil {
		return VersionResource{}, mapWriteError(err)
	}

	auditID, err := newUUID()
	if err != nil {
		return VersionResource{}, err
	}
	if err := q.InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{ID: auditID, PackageVersionID: version.ID, ActorServiceAccount: textOrNull("dev"), Action: "version.yanked", MetadataJson: []byte(`{"source":"api"}`), PackageID: version.PackageID}); err != nil {
		return VersionResource{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return VersionResource{}, err
	}

	return VersionResource{Org: req.Org, Namespace: req.Namespace, Package: req.Package, Version: result.Version, Lifecycle: result.Lifecycle, UpdatedAt: timestamptzString(result.UpdatedAt)}, nil
}

func (s *PostgresStore) UpdatePackageVisibility(ctx context.Context, org, namespace, pkg, visibility string) (PackageResource, error) {
	switch visibility {
	case "private", "internal", "public":
	default:
		return PackageResource{}, fmt.Errorf("visibility must be private, internal, or public")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PackageResource{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	row, err := q.UpdatePackageVisibility(ctx, sqlc.UpdatePackageVisibilityParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: pkg,
		Visibility:  visibility,
	})
	if err != nil {
		return PackageResource{}, mapWriteError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return PackageResource{}, err
	}

	return PackageResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		NamespaceID: uuid.UUID(row.NamespaceID.Bytes).String(),
		Org:         org,
		Namespace:   namespace,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: textValue(row.Description),
		Visibility:  row.Visibility,
		Lifecycle:   row.Lifecycle,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) CreateOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	id, err := newUUID()
	if err != nil {
		return OrgResource{}, err
	}
	row, err := s.queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		ID:          id,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Visibility:  req.Visibility,
	})
	if err != nil {
		return OrgResource{}, mapWriteError(err)
	}
	return OrgResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		Slug:        row.Slug,
		DisplayName: row.DisplayName,
		Visibility:  row.Visibility,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) EnsureOrg(ctx context.Context, req CreateOrgRequest) (OrgResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	id, err := newUUID()
	if err != nil {
		return OrgResource{}, err
	}
	row, err := s.queries.EnsureOrganization(ctx, sqlc.EnsureOrganizationParams{
		ID:          id,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Visibility:  req.Visibility,
	})
	if err != nil {
		return OrgResource{}, mapWriteError(err)
	}
	return OrgResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		Slug:        row.Slug,
		DisplayName: row.DisplayName,
		Visibility:  row.Visibility,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	id, err := newUUID()
	if err != nil {
		return NamespaceResource{}, err
	}
	row, err := s.queries.CreateNamespace(ctx, sqlc.CreateNamespaceParams{
		ID:            id,
		OrgSlug:       req.Org,
		NamespaceSlug: req.Slug,
		DisplayName:   req.DisplayName,
		Visibility:    req.Visibility,
	})
	if err != nil {
		return NamespaceResource{}, mapWriteError(err)
	}
	return NamespaceResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		OrgID:       uuid.UUID(row.OrgID.Bytes).String(),
		Slug:        row.Slug,
		DisplayName: row.DisplayName,
		Visibility:  row.Visibility,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) EnsureNamespace(ctx context.Context, req CreateNamespaceRequest) (NamespaceResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Slug
	}
	id, err := newUUID()
	if err != nil {
		return NamespaceResource{}, err
	}
	row, err := s.queries.EnsureNamespace(ctx, sqlc.EnsureNamespaceParams{
		ID:            id,
		OrgSlug:       req.Org,
		NamespaceSlug: req.Slug,
		DisplayName:   req.DisplayName,
		Visibility:    req.Visibility,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NamespaceResource{}, ErrPackageNotFound
		}
		return NamespaceResource{}, mapWriteError(err)
	}
	return NamespaceResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		OrgID:       uuid.UUID(row.OrgID.Bytes).String(),
		Slug:        row.Slug,
		DisplayName: row.DisplayName,
		Visibility:  row.Visibility,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) CreatePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PackageResource{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	id, err := newUUID()
	if err != nil {
		return PackageResource{}, err
	}
	row, err := q.CreatePackage(ctx, sqlc.CreatePackageParams{
		ID:            id,
		OrgSlug:       req.Org,
		NamespaceSlug: req.Namespace,
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		Description:   textOrNull(req.Description),
		Visibility:    req.Visibility,
	})
	if err != nil {
		return PackageResource{}, mapWriteError(err)
	}

	if req.OwnerUserID != "" {
		ownerID, err := newUUID()
		if err == nil {
			ownerUUID, parseErr := uuid.Parse(req.OwnerUserID)
			if parseErr == nil {
				if _, err := q.AddPackageMaintainer(ctx, sqlc.AddPackageMaintainerParams{
					ID:        ownerID,
					PackageID: row.ID,
					UserID:    pgtype.UUID{Bytes: ownerUUID, Valid: true},
					Role:      "owner",
				}); err != nil {
					return PackageResource{}, err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PackageResource{}, err
	}

	return PackageResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		NamespaceID: uuid.UUID(row.NamespaceID.Bytes).String(),
		Org:         req.Org,
		Namespace:   req.Namespace,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: textValue(row.Description),
		Visibility:  row.Visibility,
		Lifecycle:   row.Lifecycle,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
}

func (s *PostgresStore) EnsurePackage(ctx context.Context, req CreatePackageRequest) (PackageResource, error) {
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PackageResource{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	id, err := newUUID()
	if err != nil {
		return PackageResource{}, err
	}
	row, err := q.EnsurePackage(ctx, sqlc.EnsurePackageParams{
		ID:            id,
		OrgSlug:       req.Org,
		NamespaceSlug: req.Namespace,
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		Description:   textOrNull(req.Description),
		Visibility:    req.Visibility,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PackageResource{}, ErrPackageNotFound
		}
		return PackageResource{}, mapWriteError(err)
	}

	if req.OwnerUserID != "" {
		ownerID, err := newUUID()
		if err == nil {
			ownerUUID, parseErr := uuid.Parse(req.OwnerUserID)
			if parseErr == nil {
				if err := q.EnsurePackageMaintainer(ctx, sqlc.EnsurePackageMaintainerParams{
					ID:        ownerID,
					PackageID: row.ID,
					UserID:    pgtype.UUID{Bytes: ownerUUID, Valid: true},
					Role:      "owner",
				}); err != nil {
					return PackageResource{}, err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PackageResource{}, err
	}

	return PackageResource{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		NamespaceID: uuid.UUID(row.NamespaceID.Bytes).String(),
		Org:         req.Org,
		Namespace:   req.Namespace,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: textValue(row.Description),
		Visibility:  row.Visibility,
		Lifecycle:   row.Lifecycle,
		CreatedAt:   timestamptzString(row.CreatedAt),
		UpdatedAt:   timestamptzString(row.UpdatedAt),
	}, nil
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
		row, err := s.queries.ResolveLatest(ctx, sqlc.ResolveLatestParams{Org: org, Namespace: namespace, PackageName: name})
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

func (s *PostgresStore) GetArchive(ctx context.Context, org string, namespace string, name string, version string, format ArchiveFormat) (Archive, error) {
	manifestResult, err := s.GetManifest(ctx, org, namespace, name, version)
	if err != nil {
		return Archive{}, err
	}
	var parsed manifest.Manifest
	if err := json.Unmarshal(manifestResult.Manifest, &parsed); err != nil {
		return Archive{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}

	rows, err := s.queries.ListArchiveArtifacts(ctx, sqlc.ListArchiveArtifactsParams{Org: org, Namespace: namespace, PackageName: name, Version: strings.TrimPrefix(version, "v")})
	if err != nil {
		return Archive{}, err
	}
	byPath := map[string][]byte{}
	for _, row := range rows {
		byPath[row.Path] = row.Content
	}
	files := make([]archives.File, 0, len(parsed.Spec.Artifacts))
	for _, artifact := range parsed.Spec.Artifacts {
		content, ok := byPath[artifact.Path]
		if !ok {
			return Archive{}, ErrArtifactNotFound
		}
		files = append(files, archives.File{Path: artifact.Path, Content: content})
	}

	content, contentType, err := generateArchive(format, files)
	if err != nil {
		return Archive{}, err
	}
	cacheControl := "private, max-age=31536000, immutable"
	return Archive{ContentType: contentType, ETag: manifestResult.Digest, CacheControl: cacheControl, Content: content}, nil
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
		items = append(items, packageSummaryFromRow(row.Org, row.Namespace, row.PackageName, row.DisplayName, row.Description, row.Visibility, row.Lifecycle, row.LatestVersion))
	}

	var nextCursor *string
	if len(items) > limit {
		next := items[limit-1].Org + "/" + items[limit-1].Namespace + "/" + items[limit-1].Name
		nextCursor = &next
		items = items[:limit]
	}

	return ListPackagesResult{Items: items, NextCursor: nextCursor}, nil
}

func (s *PostgresStore) SearchPackages(ctx context.Context, params SearchParams) (SearchResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.queries.SearchPackages(ctx, sqlc.SearchPackagesParams{
		Query:        params.Query,
		Org:          params.Org,
		Namespace:    params.Namespace,
		ArtifactType: params.ArtifactType,
		Tool:         params.Tool,
		Cursor:       params.Cursor,
		PageLimit:    int32(limit + 1),
	})
	if err != nil {
		return SearchResult{}, err
	}

	items := make([]PackageSummary, 0, min(len(rows), limit))
	for _, row := range rows {
		items = append(items, packageSummaryFromRow(row.Org, row.Namespace, row.PackageName, row.DisplayName, row.Description, row.Visibility, row.Lifecycle, row.LatestVersion))
	}

	var nextCursor *string
	if len(items) > limit {
		next := items[limit-1].Org + "/" + items[limit-1].Namespace + "/" + items[limit-1].Name
		nextCursor = &next
		items = items[:limit]
	}

	return SearchResult{Items: items, NextCursor: nextCursor}, nil
}

func (s *PostgresStore) GetPackageAdoption(ctx context.Context, org string, namespace string, name string) (PackageAdoption, error) {
	adoption, err := s.queries.GetPackageAdoption(ctx, sqlc.GetPackageAdoptionParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: name,
	})
	if err != nil {
		return PackageAdoption{}, mapReadError(err)
	}

	versionRows, err := s.queries.ListPackageVersionsByAdoption(ctx, sqlc.ListPackageVersionsByAdoptionParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: name,
	})
	if err != nil {
		return PackageAdoption{}, err
	}

	versions := make([]AdoptionVersionSummary, 0, len(versionRows))
	for _, row := range versionRows {
		versions = append(versions, AdoptionVersionSummary{
			Version:      row.Version,
			InstallCount: row.InstallCount,
		})
	}

	return PackageAdoption{
		ProjectCount: adoption.ProjectCount,
		VersionCount: adoption.VersionCount,
		Versions:     versions,
	}, nil
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
			PublishedAt: version.PublishedAt,
		})
	}

	return PackageDetail{
		PackageSummary: packageSummaryFromRow(row.Org, row.Namespace, row.PackageName, row.DisplayName, row.Description, row.Visibility, row.Lifecycle, row.LatestVersion),
		Versions:       versions,
	}, nil
}

func (s *PostgresStore) CreateProject(ctx context.Context, req CreateProjectRequest) (ProjectResource, error) {
	id, err := newUUID()
	if err != nil {
		return ProjectResource{}, err
	}
	row, err := s.queries.CreateProjectRegistration(ctx, sqlc.CreateProjectRegistrationParams{
		ID:           id,
		OrgSlug:      req.Org,
		Name:         req.Name,
		RepoUrl:      req.RepoURL,
		TeamID:       pgtype.UUID{},
		MetadataJson: []byte(`{}`),
	})
	if err != nil {
		return ProjectResource{}, mapWriteError(err)
	}
	return ProjectResource{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		Org:       req.Org,
		Name:      row.Name,
		RepoURL:   row.RepoUrl,
		CreatedAt: timestamptzString(row.CreatedAt),
	}, nil
}

func (s *PostgresStore) ReportProjectAdoption(ctx context.Context, req ReportProjectAdoptionRequest) error {
	project, err := s.queries.GetProjectByRepoURL(ctx, sqlc.GetProjectByRepoURLParams{
		OrgSlug: req.Org,
		RepoUrl: req.RepoURL,
	})
	if err != nil {
		return mapReadError(err)
	}

	for _, installed := range req.Installed {
		parts := strings.SplitN(installed.Package, "/", 3)
		if len(parts) != 3 {
			continue
		}
		org, namespace, packageName := parts[0], parts[1], parts[2]

		packageID, err := s.queries.GetPackageIDForProjectInstall(ctx, sqlc.GetPackageIDForProjectInstallParams{
			Org:         org,
			Namespace:   namespace,
			PackageName: packageName,
		})
		if err != nil {
			continue
		}

		versionID, err := s.queries.GetPackageVersionIDForProjectInstall(ctx, sqlc.GetPackageVersionIDForProjectInstallParams{
			Org:         org,
			Namespace:   namespace,
			PackageName: packageName,
			Version:     strings.TrimPrefix(installed.Version, "v"),
		})
		if err != nil {
			continue
		}

		installID, err := newUUID()
		if err != nil {
			return err
		}
		if err := s.queries.UpsertProjectArtifactInstall(ctx, sqlc.UpsertProjectArtifactInstallParams{
			ID:               installID,
			ProjectID:        project.ID,
			PackageID:        packageID,
			PackageVersionID: versionID,
			InstalledDigest:  installed.Digest,
			MetadataJson:     []byte(`{}`),
		}); err != nil {
			return err
		}
	}

	return nil
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

func packageSummaryFromRow(org string, namespace string, name string, displayName string, description string, visibility string, lifecycle string, latestVersion string) PackageSummary {
	return PackageSummary{
		Org:           org,
		Namespace:     namespace,
		Name:          name,
		DisplayName:   displayName,
		Description:   description,
		Visibility:    visibility,
		Lifecycle:     lifecycle,
		LatestVersion: latestVersion,
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

func mapVersionReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrVersionNotFound
	}
	return err
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrVersionExists
	}
	if errors.As(err, &pgErr) && pgErr.Code == "23502" && (pgErr.ColumnName == "org_id" || pgErr.ColumnName == "namespace_id") {
		return ErrPackageNotFound
	}
	return err
}

func buildSearchText(parsed manifest.Manifest, org, namespace, name string, artifactRows []sqlc.ListArtifactMetadataForVersionRow) string {
	parts := []string{org, namespace, name, parsed.Metadata.DisplayName, parsed.Metadata.Description}
	for _, artifact := range parsed.Spec.Artifacts {
		parts = append(parts, artifact.Path, artifact.Type)
	}
	for _, row := range artifactRows {
		parts = append(parts, row.Path)
	}
	for label, value := range parsed.Metadata.Labels {
		parts = append(parts, label, fmt.Sprintf("%v", value))
	}
	return strings.Join(parts, " ")
}

func (s *PostgresStore) ListMaintainers(ctx context.Context, org, namespace, pkg string) ([]MaintainerResource, error) {
	packageID, err := s.queries.GetPackageID(ctx, sqlc.GetPackageIDParams{Org: org, Namespace: namespace, PackageName: pkg})
	if err != nil {
		return nil, mapReadError(err)
	}
	rows, err := s.queries.ListPackageMaintainers(ctx, packageID)
	if err != nil {
		return nil, err
	}
	res := make([]MaintainerResource, len(rows))
	for i, row := range rows {
		res[i] = MaintainerResource{
			UserID:      uuid.UUID(row.UserID.Bytes).String(),
			DisplayName: row.DisplayName,
			Email:       row.Email,
			Role:        row.Role,
		}
	}
	return res, nil
}

func (s *PostgresStore) AddMaintainer(ctx context.Context, org, namespace, pkg, userID, role string) error {
	packageID, err := s.queries.GetPackageID(ctx, sqlc.GetPackageIDParams{Org: org, Namespace: namespace, PackageName: pkg})
	if err != nil {
		return mapReadError(err)
	}
	id, err := newUUID()
	if err != nil {
		return err
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	_, err = s.queries.AddPackageMaintainer(ctx, sqlc.AddPackageMaintainerParams{
		ID:        id,
		PackageID: packageID,
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		Role:      role,
	})
	return err
}

func (s *PostgresStore) RemoveMaintainer(ctx context.Context, org, namespace, pkg, userID string) error {
	packageID, err := s.queries.GetPackageID(ctx, sqlc.GetPackageIDParams{Org: org, Namespace: namespace, PackageName: pkg})
	if err != nil {
		return mapReadError(err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return s.queries.RemovePackageMaintainerByUser(ctx, sqlc.RemovePackageMaintainerByUserParams{
		PackageID: packageID,
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
	})
}

func extractArtifactTypes(parsed manifest.Manifest) []string {
	seen := map[string]bool{}
	var types []string
	for _, artifact := range parsed.Spec.Artifacts {
		if artifact.Type != "" && !seen[artifact.Type] {
			seen[artifact.Type] = true
			types = append(types, artifact.Type)
		}
	}
	return types
}

func extractToolNames(parsed manifest.Manifest) []string {
	seen := map[string]bool{}
	var names []string
	for _, tool := range parsed.Spec.Compatibility.Tools {
		if tool.Name != "" && !seen[tool.Name] {
			seen[tool.Name] = true
			names = append(names, tool.Name)
		}
	}
	return names
}
