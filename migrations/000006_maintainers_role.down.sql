drop index if exists package_maintainers_user_idx;

alter table package_maintainers
  drop constraint if exists package_maintainers_role_check,
  drop column role,
  alter column user_id drop not null,
  add column team_id uuid references teams(id),
  add constraint package_maintainers_package_id_team_id_key unique(package_id, team_id),
  add constraint package_maintainers_actor_check check (
    (user_id is not null and team_id is null) or
    (user_id is null and team_id is not null)
  );
