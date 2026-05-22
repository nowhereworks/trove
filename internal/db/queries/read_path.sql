-- name: ResolveExact :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as resolved_version,
       coalesce(v.digest, '') as digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version)
  and v.lifecycle in ('published', 'deprecated', 'yanked');

-- name: ResolveChannel :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as resolved_version,
       coalesce(v.digest, '') as digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
join channels c on c.package_id = p.id and c.package_version_id = v.id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and c.name = sqlc.arg(channel)
  and v.lifecycle in ('published', 'deprecated', 'yanked');

-- name: ResolveDigest :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as resolved_version,
       coalesce(v.digest, '') as digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.digest = sqlc.arg(digest)::text
  and v.lifecycle in ('published', 'deprecated', 'yanked');

-- name: ResolveMajor :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as resolved_version,
       coalesce(v.digest, '') as digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.semver_major = sqlc.arg(major)
  and v.lifecycle in ('published', 'deprecated', 'yanked')
order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc
limit 1;

-- name: ResolveMinor :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as resolved_version,
       coalesce(v.digest, '') as digest
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.semver_major = sqlc.arg(major)
  and v.semver_minor = sqlc.arg(minor)
  and v.lifecycle in ('published', 'deprecated', 'yanked')
order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc
limit 1;

-- name: GetManifest :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       v.version as version,
       coalesce(v.digest, '') as digest,
       v.manifest_json as manifest_json
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version)
  and v.lifecycle in ('published', 'deprecated', 'yanked');

-- name: GetRawArtifact :one
select af.path as path,
       af.content_type as content_type,
       af.blob_digest as blob_digest,
       af.size_bytes as size_bytes,
       ab.content as content,
       case
         when o.visibility = 'public' and n.visibility = 'public' and p.visibility = 'public' and v.visibility = 'public'
         then 'public, max-age=31536000, immutable'
         else 'private, max-age=31536000, immutable'
       end as cache_control
from artifact_files af
join package_versions v on v.id = af.package_version_id
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
join artifact_blobs ab on ab.digest = af.blob_digest
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version)
  and af.path = sqlc.arg(path)
  and v.lifecycle in ('published', 'deprecated', 'yanked');

-- name: ListArchiveArtifacts :many
select af.path as path,
       ab.content as content
from artifact_files af
join package_versions v on v.id = af.package_version_id
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
join artifact_blobs ab on ab.digest = af.blob_digest
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.version = sqlc.arg(version)
  and v.lifecycle in ('published', 'deprecated', 'yanked')
order by af.path;

-- name: ListPackages :many
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       p.display_name as display_name,
       coalesce(p.description, '') as description,
       p.visibility as visibility,
       p.lifecycle as lifecycle,
       coalesce(latest.version, '') as latest_version,
       coalesce(stable.version, '') as stable_version
from packages p
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
left join channels latest_channel on latest_channel.package_id = p.id and latest_channel.name = 'latest'
left join package_versions latest on latest.id = latest_channel.package_version_id
left join channels stable_channel on stable_channel.package_id = p.id and stable_channel.name = 'stable'
left join package_versions stable on stable.id = stable_channel.package_version_id
where p.lifecycle = 'active'
  and exists (select 1 from package_versions v where v.package_id = p.id and v.lifecycle = 'published')
  and ((sqlc.arg(cursor)::text = '') or ((o.slug || '/' || n.slug || '/' || p.name) > sqlc.arg(cursor)::text))
order by o.slug, n.slug, p.name
limit sqlc.arg(page_limit)::int;

-- name: GetPackageSummary :one
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       p.display_name as display_name,
       coalesce(p.description, '') as description,
       p.visibility as visibility,
       p.lifecycle as lifecycle,
       coalesce(latest.version, '') as latest_version,
       coalesce(stable.version, '') as stable_version
from packages p
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
left join channels latest_channel on latest_channel.package_id = p.id and latest_channel.name = 'latest'
left join package_versions latest on latest.id = latest_channel.package_version_id
left join channels stable_channel on stable_channel.package_id = p.id and stable_channel.name = 'stable'
left join package_versions stable on stable.id = stable_channel.package_version_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and p.lifecycle = 'active';

-- name: SearchPackages :many
select o.slug as org,
       n.slug as namespace,
       p.name as package_name,
       p.display_name as display_name,
       coalesce(p.description, '') as description,
       p.visibility as visibility,
       p.lifecycle as lifecycle,
       coalesce(latest.version, '') as latest_version,
       coalesce(stable.version, '') as stable_version,
       ts_rank(sd.search_text, plainto_tsquery('english', sqlc.arg(query)::text)) as rank
from package_search_documents sd
join packages p on p.id = sd.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
left join channels latest_channel on latest_channel.package_id = p.id and latest_channel.name = 'latest'
left join package_versions latest on latest.id = latest_channel.package_version_id
left join channels stable_channel on stable_channel.package_id = p.id and stable_channel.name = 'stable'
left join package_versions stable on stable.id = stable_channel.package_version_id
where sd.search_text @@ plainto_tsquery('english', sqlc.arg(query)::text)
  and sd.lifecycle = 'active'
  and sd.visibility = 'public'
  and (sqlc.arg(org)::text = '' or o.slug = sqlc.arg(org)::text)
  and (sqlc.arg(namespace)::text = '' or n.slug = sqlc.arg(namespace)::text)
  and (sqlc.arg(artifact_type)::text = '' or sqlc.arg(artifact_type)::text = any(sd.artifact_types))
  and (sqlc.arg(tool)::text = '' or sqlc.arg(tool)::text = any(sd.tool_names))
  and ((sqlc.arg(cursor)::text = '') or ((o.slug || '/' || n.slug || '/' || p.name) > sqlc.arg(cursor)::text))
order by rank desc, o.slug, n.slug, p.name
limit sqlc.arg(page_limit)::int;

-- name: GetPackageAdoption :one
select count(distinct pai.project_id)::bigint as project_count,
       count(distinct pai.package_version_id)::bigint as version_count
from project_artifact_installs pai
join packages p on p.id = pai.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name);

-- name: ListPackageVersionsByAdoption :many
select pv.version,
       count(pai.id)::bigint as install_count
from project_artifact_installs pai
join package_versions pv on pv.id = pai.package_version_id
join packages p on p.id = pv.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
group by pv.version
order by install_count desc, pv.semver_major desc, pv.semver_minor desc, pv.semver_patch desc;

-- name: ListPackageVersions :many
select v.version as version,
       coalesce(v.digest, '') as digest,
       v.lifecycle as lifecycle,
       coalesce(v.channel, '') as channel,
       coalesce(to_char(v.published_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')::text as published_at
from package_versions v
join packages p on p.id = v.package_id
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name)
  and v.lifecycle in ('published', 'deprecated', 'yanked')
order by v.semver_major desc, v.semver_minor desc, v.semver_patch desc;
