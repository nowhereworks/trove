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

create index channels_package_name_idx on channels(package_id, name);

alter table package_versions add column channel text;
