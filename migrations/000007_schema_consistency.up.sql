alter table project_registrations
  add constraint project_registrations_team_id_fkey
  foreign key (team_id) references teams(id);

alter table packages
  add constraint packages_lifecycle_check
  check (lifecycle in ('active', 'deprecated'));

alter table package_versions
  add constraint package_versions_lifecycle_check
  check (lifecycle in ('draft', 'review', 'published', 'deprecated', 'yanked'));

alter table package_search_documents
  add constraint package_search_documents_lifecycle_check
  check (lifecycle in ('active', 'deprecated'));
