create table package_search_documents (
  package_id uuid primary key references packages(id),
  latest_published_version_id uuid references package_versions(id),
  search_text tsvector not null,
  labels_json jsonb not null default '{}'::jsonb,
  artifact_types text[] not null default '{}',
  lifecycle text not null,
  visibility text not null,
  updated_at timestamptz not null
);

create index package_search_documents_search_idx on package_search_documents using gin(search_text);
create index package_search_documents_lifecycle_idx on package_search_documents(lifecycle);
create index package_search_documents_visibility_idx on package_search_documents(visibility);
create index package_search_documents_artifact_types_idx on package_search_documents using gin(artifact_types);

create table project_registrations (
  id uuid primary key,
  org_id uuid not null references organizations(id),
  name text not null,
  repo_url text not null,
  team_id uuid,
  metadata_json jsonb,
  last_seen_at timestamptz,
  created_at timestamptz not null,
  unique(org_id, repo_url)
);

create index project_registrations_org_idx on project_registrations(org_id);

create table project_artifact_installs (
  id uuid primary key,
  project_id uuid not null references project_registrations(id),
  package_id uuid not null references packages(id),
  package_version_id uuid not null references package_versions(id),
  installed_digest text not null,
  installed_at timestamptz not null,
  last_checked_at timestamptz,
  metadata_json jsonb
);

create index project_artifact_installs_project_idx on project_artifact_installs(project_id);
create index project_artifact_installs_package_idx on project_artifact_installs(package_id);
create index project_artifact_installs_package_version_idx on project_artifact_installs(package_version_id);
