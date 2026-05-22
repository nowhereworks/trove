create table organizations (
  id uuid primary key,
  slug text unique not null,
  display_name text not null,
  description text,
  visibility text not null default 'private',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint organizations_slug_check check (slug ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$'),
  constraint organizations_visibility_check check (visibility in ('private', 'internal', 'public'))
);

create table namespaces (
  id uuid primary key,
  org_id uuid not null references organizations(id),
  slug text not null,
  display_name text not null,
  description text,
  visibility text not null default 'private',
  owner_team_id uuid,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(org_id, slug),
  constraint namespaces_slug_check check (slug ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$'),
  constraint namespaces_visibility_check check (visibility in ('private', 'internal', 'public'))
);

create table packages (
  id uuid primary key,
  org_id uuid not null references organizations(id),
  namespace_id uuid not null references namespaces(id),
  name text not null,
  display_name text not null,
  description text,
  visibility text not null default 'private',
  lifecycle text not null default 'active',
  owner_team_id uuid,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(namespace_id, name),
  constraint packages_name_check check (name ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$'),
  constraint packages_visibility_check check (visibility in ('private', 'internal', 'public'))
);

create table package_versions (
  id uuid primary key,
  package_id uuid not null references packages(id),
  version text not null,
  semver_major int,
  semver_minor int,
  semver_patch int,
  lifecycle text not null,
  visibility text not null default 'private',
  channel text,
  manifest_json jsonb not null,
  changelog text,
  digest text,
  created_by uuid,
  published_by uuid,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  published_at timestamptz,
  unique(package_id, version),
  constraint package_versions_visibility_check check (visibility in ('private', 'internal', 'public')),
  constraint package_versions_published_semver_check check (lifecycle <> 'published' or version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
  constraint package_versions_published_digest_check check (lifecycle <> 'published' or (digest is not null and published_at is not null))
);

create table artifact_blobs (
  digest text primary key,
  content bytea not null,
  size_bytes bigint not null,
  created_at timestamptz not null
);

create table artifact_locations (
  digest text primary key references artifact_blobs(digest),
  storage_driver text not null default 'postgres',
  storage_uri text,
  created_at timestamptz not null,
  constraint artifact_locations_storage_driver_check check (storage_driver in ('postgres', 's3'))
);

create table artifact_files (
  id uuid primary key,
  package_version_id uuid not null references package_versions(id),
  path text not null,
  type text not null,
  content_type text not null,
  blob_digest text not null references artifact_blobs(digest),
  size_bytes bigint not null,
  target_path text,
  created_at timestamptz not null,
  unique(package_version_id, path),
  constraint artifact_files_path_check check (path <> '' and path not like '/%' and path not like '%..%')
);

create table channels (
  id uuid primary key,
  package_id uuid not null references packages(id),
  name text not null,
  package_version_id uuid not null references package_versions(id),
  updated_by uuid,
  updated_at timestamptz not null,
  unique(package_id, name),
  constraint channels_name_check check (name in ('latest', 'stable'))
);

create table audit_events (
  id uuid primary key,
  org_id uuid references organizations(id),
  namespace_id uuid references namespaces(id),
  package_id uuid references packages(id),
  package_version_id uuid references package_versions(id),
  actor_user_id uuid,
  actor_service_account text,
  action text not null,
  metadata_json jsonb not null default '{}'::jsonb,
  created_at timestamptz not null
);

create index packages_namespace_name_idx on packages(namespace_id, name);
create index package_versions_package_version_idx on package_versions(package_id, version);
create index package_versions_package_lifecycle_idx on package_versions(package_id, lifecycle);
create index package_versions_package_semver_idx on package_versions(package_id, semver_major, semver_minor, semver_patch);
create index artifact_files_version_path_idx on artifact_files(package_version_id, path);
create index channels_package_name_idx on channels(package_id, name);

create or replace function prevent_published_version_update()
returns trigger language plpgsql as $$
begin
  if old.lifecycle = 'published' and (
    old.version is distinct from new.version or
    old.manifest_json is distinct from new.manifest_json or
    old.digest is distinct from new.digest
  ) then
    raise exception 'published package versions are immutable';
  end if;
  return new;
end;
$$;

create trigger package_versions_immutable_update
before update on package_versions
for each row execute function prevent_published_version_update();

create or replace function prevent_published_artifact_change()
returns trigger language plpgsql as $$
declare
  version_lifecycle text;
begin
  select lifecycle into version_lifecycle
  from package_versions
  where id = coalesce(new.package_version_id, old.package_version_id);

  if version_lifecycle = 'published' then
    raise exception 'artifacts for published package versions are immutable';
  end if;

  return coalesce(new, old);
end;
$$;

create trigger artifact_files_immutable_insert
before insert on artifact_files
for each row execute function prevent_published_artifact_change();

create trigger artifact_files_immutable_update
before update on artifact_files
for each row execute function prevent_published_artifact_change();

create trigger artifact_files_immutable_delete
before delete on artifact_files
for each row execute function prevent_published_artifact_change();
