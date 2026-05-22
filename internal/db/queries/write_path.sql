-- name: GetPackageID :one
select p.id
from packages p
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name);

-- name: CreateDraftVersion :one
insert into package_versions (
  id,
  package_id,
  version,
  semver_major,
  semver_minor,
  semver_patch,
  lifecycle,
  visibility,
  manifest_json,
  created_at,
  updated_at
)
values (
  sqlc.arg(id),
  sqlc.arg(package_id),
  sqlc.arg(version),
  sqlc.arg(semver_major),
  sqlc.arg(semver_minor),
  sqlc.arg(semver_patch),
  'draft',
  sqlc.arg(visibility),
  '{}'::jsonb,
  now(),
  now()
)
returning id, version, lifecycle, visibility, digest, created_at, updated_at, published_at;

-- name: GetVersionForUpdate :one
select v.id,
       v.package_id,
       v.version,
       v.lifecycle,
       v.visibility,
       v.manifest_json,
       v.digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version)
for update;

-- name: UpsertArtifactBlob :exec
insert into artifact_blobs (digest, content, size_bytes, created_at)
values (sqlc.arg(digest), sqlc.arg(content), sqlc.arg(size_bytes), now())
on conflict (digest) do nothing;

-- name: UpsertArtifactLocation :exec
insert into artifact_locations (digest, storage_driver, storage_uri, created_at)
values (sqlc.arg(digest), 'postgres', null, now())
on conflict (digest) do nothing;

-- name: UpsertArtifactFile :one
insert into artifact_files (id, package_version_id, path, type, content_type, blob_digest, size_bytes, target_path, created_at)
values (sqlc.arg(id), sqlc.arg(package_version_id), sqlc.arg(path), sqlc.arg(type), sqlc.arg(content_type), sqlc.arg(blob_digest), sqlc.arg(size_bytes), sqlc.arg(target_path), now())
on conflict (package_version_id, path) do update
set type = excluded.type,
    content_type = excluded.content_type,
    blob_digest = excluded.blob_digest,
    size_bytes = excluded.size_bytes,
    target_path = excluded.target_path
returning id, path, type, content_type, blob_digest, size_bytes, target_path, created_at;

-- name: UpdateVersionManifest :exec
update package_versions
set manifest_json = sqlc.arg(manifest_json),
    visibility = sqlc.arg(visibility),
    channel = sqlc.arg(channel),
    updated_at = now()
where id = sqlc.arg(id);

-- name: ListArtifactMetadataForVersion :many
select path, type, blob_digest, size_bytes, coalesce(target_path, '') as target_path
from artifact_files
where package_version_id = sqlc.arg(package_version_id)
order by path;

-- name: PublishVersion :one
update package_versions
set lifecycle = 'published',
    digest = sqlc.arg(digest),
    updated_at = now(),
    published_at = now()
where id = sqlc.arg(id)
returning version, lifecycle, digest, published_at;

-- name: UpsertChannel :exec
insert into channels (id, package_id, name, package_version_id, updated_at)
values (sqlc.arg(id), sqlc.arg(package_id), sqlc.arg(name), sqlc.arg(package_version_id), now())
on conflict (package_id, name) do update
set package_version_id = excluded.package_version_id,
    updated_at = excluded.updated_at;

-- name: InsertAuditEvent :exec
insert into audit_events (id, org_id, namespace_id, package_id, package_version_id, actor_service_account, action, metadata_json, created_at)
select sqlc.arg(id), p.org_id, p.namespace_id, p.id, sqlc.arg(package_version_id), sqlc.arg(actor_service_account), sqlc.arg(action), sqlc.arg(metadata_json), now()
from packages p
where p.id = sqlc.arg(package_id);
