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

-- name: GetPackageVersionAny :one
select v.version,
       v.lifecycle,
       v.visibility,
       coalesce(v.digest, '') as digest,
       coalesce(to_char(v.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as created_at,
       coalesce(to_char(v.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as updated_at,
       coalesce(to_char(v.published_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as published_at
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version);

-- name: ResetUnpublishedVersionToDraft :one
with version_row as (
  select v.id
  from package_versions v
  join packages p on p.id = v.package_id
  join namespaces n on n.id = p.namespace_id
  join organizations o on o.id = n.org_id
  where o.slug = sqlc.arg(org)
    and n.slug = sqlc.arg(namespace)
    and p.name = sqlc.arg(package_name)
    and v.version = sqlc.arg(version)
    and v.lifecycle in ('draft', 'review')
), deleted_approvals as (
  delete from approvals
  where package_version_id in (select id from version_row)
), deleted_comments as (
  delete from review_comments
  where review_id in (select id from reviews where package_version_id in (select id from version_row))
), deleted_reviews as (
  delete from reviews
  where package_version_id in (select id from version_row)
)
update package_versions
set lifecycle = 'draft',
    digest = null,
    published_at = null,
    updated_at = now()
where id in (select id from version_row)
returning version, lifecycle, visibility, coalesce(digest, '') as digest,
          coalesce(to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as created_at,
          coalesce(to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as updated_at,
          coalesce(to_char(published_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as published_at;

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

-- name: InsertAuditEvent :exec
insert into audit_events (id, org_id, namespace_id, package_id, package_version_id, actor_service_account, action, metadata_json, created_at)
select sqlc.arg(id), p.org_id, p.namespace_id, p.id, sqlc.arg(package_version_id), sqlc.arg(actor_service_account), sqlc.arg(action), sqlc.arg(metadata_json), now()
from packages p
where p.id = sqlc.arg(package_id);

-- name: DeprecateVersion :one
update package_versions
set lifecycle = 'deprecated',
    updated_at = now()
where id = sqlc.arg(id)
  and lifecycle = 'published'
returning version, lifecycle, updated_at;

-- name: YankVersion :one
update package_versions
set lifecycle = 'yanked',
    updated_at = now()
where id = sqlc.arg(id)
  and lifecycle in ('published', 'deprecated')
returning version, lifecycle, updated_at;

-- name: CreateOrganization :one
insert into organizations (id, slug, display_name, visibility, created_at, updated_at)
values (sqlc.arg(id), sqlc.arg(slug), sqlc.arg(display_name), sqlc.arg(visibility), now(), now())
returning id, slug, display_name, visibility, created_at, updated_at;

-- name: EnsureOrganization :one
insert into organizations (id, slug, display_name, visibility, created_at, updated_at)
values (sqlc.arg(id), sqlc.arg(slug), sqlc.arg(display_name), sqlc.arg(visibility), now(), now())
on conflict (slug) do update set slug = organizations.slug
returning id, slug, display_name, visibility, created_at, updated_at;

-- name: CreateNamespace :one
insert into namespaces (id, org_id, slug, display_name, visibility, created_at, updated_at)
values (
  sqlc.arg(id),
  (select o.id from organizations o where o.slug = sqlc.arg(org_slug)),
  sqlc.arg(namespace_slug),
  sqlc.arg(display_name),
  sqlc.arg(visibility),
  now(),
  now()
)
returning id, org_id, slug, display_name, visibility, created_at, updated_at;

-- name: EnsureNamespace :one
with org_row as (
  select id from organizations where organizations.slug = sqlc.arg(org_slug)
)
insert into namespaces (id, org_id, slug, display_name, visibility, created_at, updated_at)
select
  sqlc.arg(id),
  org_row.id,
  sqlc.arg(namespace_slug),
  sqlc.arg(display_name),
  sqlc.arg(visibility),
  now(),
  now()
from org_row
on conflict (org_id, slug) do update set slug = namespaces.slug
returning id, org_id, slug, display_name, visibility, created_at, updated_at;

-- name: CreatePackage :one
insert into packages (id, org_id, namespace_id, name, display_name, description, visibility, lifecycle, created_at, updated_at)
values (
  sqlc.arg(id),
  (select o.id from organizations o where o.slug = sqlc.arg(org_slug)),
  (select n.id from namespaces n join organizations o on o.id = n.org_id where o.slug = sqlc.arg(org_slug) and n.slug = sqlc.arg(namespace_slug)),
  sqlc.arg(name),
  sqlc.arg(display_name),
  sqlc.arg(description),
  sqlc.arg(visibility),
  'active',
  now(),
  now()
)
returning id, namespace_id, name, display_name, description, visibility, lifecycle, created_at, updated_at;

-- name: EnsurePackage :one
with namespace_row as (
  select o.id as org_id, n.id as namespace_id
  from namespaces n
  join organizations o on o.id = n.org_id
  where o.slug = sqlc.arg(org_slug)
    and n.slug = sqlc.arg(namespace_slug)
)
insert into packages (id, org_id, namespace_id, name, display_name, description, visibility, lifecycle, created_at, updated_at)
select
  sqlc.arg(id),
  namespace_row.org_id,
  namespace_row.namespace_id,
  sqlc.arg(name),
  sqlc.arg(display_name),
  sqlc.arg(description),
  sqlc.arg(visibility),
  'active',
  now(),
  now()
from namespace_row
on conflict (namespace_id, name) do update set name = packages.name
returning id, namespace_id, name, display_name, description, visibility, lifecycle, created_at, updated_at;

-- name: UpsertSearchDocument :exec
insert into package_search_documents (
  package_id,
  latest_published_version_id,
  search_text,
  labels_json,
  artifact_types,
  tool_names,
  lifecycle,
  visibility,
  updated_at
)
values (
  sqlc.arg(package_id),
  sqlc.arg(latest_published_version_id),
  to_tsvector('english', sqlc.arg(search_text)),
  sqlc.arg(labels_json),
  sqlc.arg(artifact_types)::text[],
  sqlc.arg(tool_names)::text[],
  sqlc.arg(lifecycle),
  sqlc.arg(visibility),
  now()
)
on conflict (package_id) do update
set latest_published_version_id = excluded.latest_published_version_id,
    search_text = excluded.search_text,
    labels_json = excluded.labels_json,
    artifact_types = excluded.artifact_types,
    tool_names = excluded.tool_names,
    lifecycle = excluded.lifecycle,
    visibility = excluded.visibility,
    updated_at = excluded.updated_at;

-- name: CreateProjectRegistration :one
insert into project_registrations (id, org_id, name, repo_url, team_id, metadata_json, last_seen_at, created_at)
values (
  sqlc.arg(id),
  (select o.id from organizations o where o.slug = sqlc.arg(org_slug)),
  sqlc.arg(name),
  sqlc.arg(repo_url),
  sqlc.arg(team_id),
  sqlc.arg(metadata_json),
  now(),
  now()
)
on conflict (org_id, repo_url) do update
set name = excluded.name,
    team_id = excluded.team_id,
    metadata_json = excluded.metadata_json,
    last_seen_at = now()
returning id, org_id, name, repo_url, created_at;

-- name: GetProjectByRepoURL :one
select pr.id, pr.org_id, pr.name, pr.repo_url, pr.team_id, pr.metadata_json, pr.last_seen_at, pr.created_at
from project_registrations pr
join organizations o on o.id = pr.org_id
where o.slug = sqlc.arg(org_slug)
  and pr.repo_url = sqlc.arg(repo_url);

-- name: UpsertProjectArtifactInstall :exec
insert into project_artifact_installs (id, project_id, package_id, package_version_id, installed_digest, installed_at, last_checked_at, metadata_json)
values (
  sqlc.arg(id),
  sqlc.arg(project_id),
  sqlc.arg(package_id),
  sqlc.arg(package_version_id),
  sqlc.arg(installed_digest),
  now(),
  now(),
  sqlc.arg(metadata_json)
)
on conflict (project_id, package_id) do update
set package_version_id = excluded.package_version_id,
    installed_digest = excluded.installed_digest,
    installed_at = excluded.installed_at,
    last_checked_at = now(),
    metadata_json = excluded.metadata_json;

-- name: GetPackageIDForProjectInstall :one
select p.id
from packages p
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name);

-- name: GetPackageVersionIDForProjectInstall :one
select v.id
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version);
