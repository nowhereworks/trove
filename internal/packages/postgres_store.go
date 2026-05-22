package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Resolve(ctx context.Context, org string, namespace string, name string, selector string) (ResolvedVersion, error) {
	parsed, err := ParseSelector(selector)
	if err != nil {
		return ResolvedVersion{}, err
	}

	var row pgx.Row
	switch parsed.Kind {
	case SelectorExact:
		row = s.pool.QueryRow(ctx, resolvedVersionSQL+` and v.version = $4`, org, namespace, name, parsed.Version)
	case SelectorChannel:
		row = s.pool.QueryRow(ctx, resolvedVersionSQL+` and c.name = $4`, org, namespace, name, parsed.Value)
	case SelectorDigest:
		row = s.pool.QueryRow(ctx, resolvedVersionSQL+` and v.digest = $4`, org, namespace, name, parsed.Value)
	case SelectorMajor:
		row = s.pool.QueryRow(ctx, resolvedVersionSQL+` and v.semver_major = $4 order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc limit 1`, org, namespace, name, parsed.Major)
	case SelectorMinor:
		row = s.pool.QueryRow(ctx, resolvedVersionSQL+` and v.semver_major = $4 and v.semver_minor = $5 order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc limit 1`, org, namespace, name, parsed.Major, parsed.Minor)
	default:
		return ResolvedVersion{}, ErrInvalidSelector
	}

	var result ResolvedVersion
	if err := row.Scan(&result.Org, &result.Namespace, &result.Package, &result.ResolvedVersion, &result.Digest); err != nil {
		return ResolvedVersion{}, mapReadError(err)
	}
	result.Selector = selector
	return FillResolvedURLs(result), nil
}

func (s *PostgresStore) GetManifest(ctx context.Context, org string, namespace string, name string, version string) (Manifest, error) {
	version = strings.TrimPrefix(version, "v")
	var result Manifest
	var manifestBytes []byte
	err := s.pool.QueryRow(ctx, `
		select o.slug, n.slug, p.name, v.version, v.digest, v.manifest_json
		from package_versions v
		join packages p on p.id = v.package_id
		join namespaces n on n.id = p.namespace_id
		join organizations o on o.id = n.org_id
		where o.slug = $1
		  and n.slug = $2
		  and p.name = $3
		  and v.version = $4
		  and v.lifecycle in ('published', 'deprecated', 'yanked')
	`, org, namespace, name, version).Scan(&result.Org, &result.Namespace, &result.Package, &result.Version, &result.Digest, &manifestBytes)
	if err != nil {
		return Manifest{}, mapReadError(err)
	}
	result.Manifest = json.RawMessage(manifestBytes)
	return result, nil
}

func (s *PostgresStore) GetRawArtifact(ctx context.Context, org string, namespace string, name string, version string, path string) (RawArtifact, error) {
	version = strings.TrimPrefix(version, "v")
	var result RawArtifact
	err := s.pool.QueryRow(ctx, `
		select af.path,
		       af.content_type,
		       af.blob_digest,
		       af.size_bytes,
		       ab.content,
		       case
		         when o.visibility = 'public' and n.visibility = 'public' and p.visibility = 'public' and v.visibility = 'public'
		         then 'public, max-age=31536000, immutable'
		         else 'private, max-age=31536000, immutable'
		       end
		from artifact_files af
		join package_versions v on v.id = af.package_version_id
		join packages p on p.id = v.package_id
		join namespaces n on n.id = p.namespace_id
		join organizations o on o.id = n.org_id
		join artifact_blobs ab on ab.digest = af.blob_digest
		where o.slug = $1
		  and n.slug = $2
		  and p.name = $3
		  and v.version = $4
		  and af.path = $5
		  and v.lifecycle in ('published', 'deprecated', 'yanked')
	`, org, namespace, name, version, path).Scan(
		&result.Path,
		&result.ContentType,
		&result.BlobDigest,
		&result.SizeBytes,
		&result.Content,
		&result.CacheControl,
	)
	if err != nil {
		return RawArtifact{}, mapReadError(err)
	}
	return result, nil
}

func (s *PostgresStore) ListPackages(ctx context.Context, params ListPackagesParams) (ListPackagesResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		select o.slug,
		       n.slug,
		       p.name,
		       p.display_name,
		       coalesce(p.description, ''),
		       p.visibility,
		       p.lifecycle,
		       coalesce(latest.version, ''),
		       coalesce(stable.version, '')
		from packages p
		join namespaces n on n.id = p.namespace_id
		join organizations o on o.id = n.org_id
		left join channels latest_channel on latest_channel.package_id = p.id and latest_channel.name = 'latest'
		left join package_versions latest on latest.id = latest_channel.package_version_id
		left join channels stable_channel on stable_channel.package_id = p.id and stable_channel.name = 'stable'
		left join package_versions stable on stable.id = stable_channel.package_version_id
		where p.lifecycle = 'active'
		  and exists (select 1 from package_versions v where v.package_id = p.id and v.lifecycle = 'published')
		  and (($1::text = '') or ((o.slug || '/' || n.slug || '/' || p.name) > $1))
		order by o.slug, n.slug, p.name
		limit $2
	`, params.Cursor, limit+1)
	if err != nil {
		return ListPackagesResult{}, fmt.Errorf("list packages: %w", err)
	}
	defer rows.Close()

	items := make([]PackageSummary, 0, limit)
	for rows.Next() {
		var item PackageSummary
		if err := rows.Scan(&item.Org, &item.Namespace, &item.Name, &item.DisplayName, &item.Description, &item.Visibility, &item.Lifecycle, &item.LatestVersion, &item.StableVersion); err != nil {
			return ListPackagesResult{}, fmt.Errorf("scan package: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ListPackagesResult{}, fmt.Errorf("read packages: %w", err)
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
	var summary PackageSummary
	err := s.pool.QueryRow(ctx, `
		select o.slug,
		       n.slug,
		       p.name,
		       p.display_name,
		       coalesce(p.description, ''),
		       p.visibility,
		       p.lifecycle,
		       coalesce(latest.version, ''),
		       coalesce(stable.version, '')
		from packages p
		join namespaces n on n.id = p.namespace_id
		join organizations o on o.id = n.org_id
		left join channels latest_channel on latest_channel.package_id = p.id and latest_channel.name = 'latest'
		left join package_versions latest on latest.id = latest_channel.package_version_id
		left join channels stable_channel on stable_channel.package_id = p.id and stable_channel.name = 'stable'
		left join package_versions stable on stable.id = stable_channel.package_version_id
		where o.slug = $1 and n.slug = $2 and p.name = $3 and p.lifecycle = 'active'
	`, org, namespace, name).Scan(&summary.Org, &summary.Namespace, &summary.Name, &summary.DisplayName, &summary.Description, &summary.Visibility, &summary.Lifecycle, &summary.LatestVersion, &summary.StableVersion)
	if err != nil {
		return PackageDetail{}, mapReadError(err)
	}

	rows, err := s.pool.Query(ctx, `
		select v.version,
		       v.digest,
		       v.lifecycle,
		       coalesce(v.channel, ''),
		       coalesce(to_char(v.published_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from package_versions v
		join packages p on p.id = v.package_id
		join namespaces n on n.id = p.namespace_id
		join organizations o on o.id = n.org_id
		where o.slug = $1 and n.slug = $2 and p.name = $3
		  and v.lifecycle in ('published', 'deprecated', 'yanked')
		order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc
	`, org, namespace, name)
	if err != nil {
		return PackageDetail{}, fmt.Errorf("query package versions: %w", err)
	}
	defer rows.Close()

	versions := []PackageVersionSummary{}
	for rows.Next() {
		var version PackageVersionSummary
		if err := rows.Scan(&version.Version, &version.Digest, &version.Lifecycle, &version.Channel, &version.PublishedAt); err != nil {
			return PackageDetail{}, fmt.Errorf("scan package version: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return PackageDetail{}, fmt.Errorf("read package versions: %w", err)
	}

	return PackageDetail{PackageSummary: summary, Versions: versions}, nil
}

const resolvedVersionSQL = `
	select o.slug, n.slug, p.name, v.version, v.digest
	from package_versions v
	join packages p on p.id = v.package_id
	join namespaces n on n.id = p.namespace_id
	join organizations o on o.id = n.org_id
	left join channels c on c.package_id = p.id and c.package_version_id = v.id
	where o.slug = $1
	  and n.slug = $2
	  and p.name = $3
	  and v.lifecycle in ('published', 'deprecated', 'yanked')
`

func mapReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPackageNotFound
	}
	return err
}
