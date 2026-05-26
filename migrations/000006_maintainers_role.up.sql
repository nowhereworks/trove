delete from package_maintainers where user_id is null;

alter table package_maintainers
  drop constraint if exists package_maintainers_actor_check,
  drop constraint if exists package_maintainers_team_id_key,
  drop column team_id,
  alter column user_id set not null,
  add column role text not null default 'maintainer',
  add constraint package_maintainers_role_check check (role in ('owner', 'maintainer'));

create index package_maintainers_user_idx on package_maintainers(user_id);
