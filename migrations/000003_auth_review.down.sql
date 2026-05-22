drop index if exists api_tokens_actor_user_idx;
drop index if exists api_tokens_hash_idx;
drop index if exists approvals_package_version_idx;
drop index if exists reviews_package_version_idx;
drop index if exists package_maintainers_package_idx;
drop index if exists teams_org_slug_idx;

drop table if exists api_tokens;
drop table if exists approvals;
drop table if exists review_comments;
drop table if exists reviews;
drop table if exists package_maintainers;
drop table if exists team_memberships;
drop table if exists teams;
drop table if exists users;
