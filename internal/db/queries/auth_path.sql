-- name: GetUserByOIDC :one
select id, email, display_name, avatar_url, oidc_issuer, oidc_subject, created_at, updated_at
from users
where oidc_issuer = sqlc.arg(oidc_issuer)
  and oidc_subject = sqlc.arg(oidc_subject);

-- name: CreateUser :one
insert into users (id, email, display_name, oidc_issuer, oidc_subject, created_at, updated_at)
values (sqlc.arg(id), sqlc.arg(email), sqlc.arg(display_name), sqlc.arg(oidc_issuer), sqlc.arg(oidc_subject), now(), now())
returning id, email, display_name, avatar_url, oidc_issuer, oidc_subject, created_at, updated_at;

-- name: GetUserByID :one
select id, email, display_name, avatar_url, oidc_issuer, oidc_subject, created_at, updated_at
from users
where id = sqlc.arg(id);

-- name: GetAPITokenByHash :one
select id, token_hash, display_name, actor_user_id, actor_service_account, scopes,
       org_id, namespace_id, package_id, created_at, expires_at, last_used_at, revoked_at
from api_tokens
where token_hash = sqlc.arg(token_hash);

-- name: CreateAPIToken :one
insert into api_tokens (id, token_hash, display_name, actor_user_id, actor_service_account, scopes,
                        org_id, namespace_id, package_id, created_at, expires_at)
values (sqlc.arg(id), sqlc.arg(token_hash), sqlc.arg(display_name), sqlc.arg(actor_user_id),
        sqlc.arg(actor_service_account), sqlc.arg(scopes)::text[], sqlc.arg(org_id),
        sqlc.arg(namespace_id), sqlc.arg(package_id), now(), sqlc.arg(expires_at))
returning id, token_hash, display_name, actor_user_id, actor_service_account, scopes,
          org_id, namespace_id, package_id, created_at, expires_at, last_used_at, revoked_at;

-- name: RevokeAPIToken :exec
update api_tokens
set revoked_at = now()
where id = sqlc.arg(id) and revoked_at is null;

-- name: UpdateTokenLastUsed :exec
update api_tokens
set last_used_at = now()
where id = sqlc.arg(id) and revoked_at is null;

-- name: IsPackageMaintainer :one
select exists (
  select 1 from package_maintainers pm
  where pm.package_id = sqlc.arg(package_id)
    and pm.user_id = sqlc.arg(user_id)
) as is_maintainer;

-- name: IsPackageOwner :one
select exists (
  select 1 from package_maintainers pm
  where pm.package_id = sqlc.arg(package_id)
    and pm.user_id = sqlc.arg(user_id)
    and pm.role = 'owner'
) as is_owner;

-- name: ListPackageMaintainers :many
select pm.id, pm.package_id, pm.user_id, pm.role, pm.created_at,
       u.display_name, u.email
from package_maintainers pm
join users u on u.id = pm.user_id
where pm.package_id = sqlc.arg(package_id)
order by pm.role, u.display_name;

-- name: AddPackageMaintainer :one
insert into package_maintainers (id, package_id, user_id, role, created_at)
values (sqlc.arg(id), sqlc.arg(package_id), sqlc.arg(user_id), sqlc.arg(role), now())
returning id, package_id, user_id, role, created_at;

-- name: EnsurePackageMaintainer :exec
insert into package_maintainers (id, package_id, user_id, role, created_at)
values (sqlc.arg(id), sqlc.arg(package_id), sqlc.arg(user_id), sqlc.arg(role), now())
on conflict (package_id, user_id) do nothing;

-- name: RemovePackageMaintainerByUser :exec
delete from package_maintainers
where package_id = sqlc.arg(package_id) and user_id = sqlc.arg(user_id);

-- name: UpdateMaintainerRole :exec
update package_maintainers
set role = sqlc.arg(role)
where package_id = sqlc.arg(package_id) and user_id = sqlc.arg(user_id);

-- name: CreateReview :one
with created_review as (
  insert into reviews (id, package_version_id, reviewer_id, status, comment, created_at, updated_at)
  select sqlc.arg(id), sqlc.arg(package_version_id), sqlc.arg(reviewer_id), sqlc.arg(status),
         sqlc.arg(comment), now(), now()
  where exists (
    select 1 from package_versions
    where id = sqlc.arg(package_version_id)
      and lifecycle in ('draft', 'review')
  )
  returning id, package_version_id, reviewer_id, status, comment, created_at, updated_at
), updated_version as (
  update package_versions
  set lifecycle = 'review', updated_at = now()
  where id = sqlc.arg(package_version_id)
    and lifecycle = 'draft'
    and exists (select 1 from created_review)
  returning id
)
select id, package_version_id, reviewer_id, status, comment, created_at, updated_at
from created_review;

-- name: GetReview :one
select id, package_version_id, reviewer_id, status, comment, created_at, updated_at
from reviews
where id = sqlc.arg(id);

-- name: ListReviewsForVersion :many
select r.id, r.package_version_id, r.reviewer_id, r.status, r.comment, r.created_at, r.updated_at
from reviews r
where r.package_version_id = sqlc.arg(package_version_id)
order by r.created_at desc;

-- name: UpdateReviewStatus :one
update reviews
set status = sqlc.arg(status), comment = coalesce(sqlc.arg(comment), comment), updated_at = now()
where id = sqlc.arg(id)
returning id, package_version_id, reviewer_id, status, comment, created_at, updated_at;

-- name: AddReviewComment :one
insert into review_comments (id, review_id, author_id, body, created_at)
values (sqlc.arg(id), sqlc.arg(review_id), sqlc.arg(author_id), sqlc.arg(body), now())
returning id, review_id, author_id, body, created_at;

-- name: ListReviewComments :many
select rc.id, rc.review_id, rc.author_id, rc.body, rc.created_at
from review_comments rc
where rc.review_id = sqlc.arg(review_id)
order by rc.created_at;

-- name: UpsertApproval :exec
insert into approvals (id, package_version_id, reviewer_id, created_at)
values (sqlc.arg(id), sqlc.arg(package_version_id), sqlc.arg(reviewer_id), now())
on conflict (package_version_id, reviewer_id) do nothing;

-- name: RemoveApproval :exec
delete from approvals
where package_version_id = sqlc.arg(package_version_id) and reviewer_id = sqlc.arg(reviewer_id);

-- name: CountApprovalsForVersion :one
select count(*) as approval_count
from approvals
where package_version_id = sqlc.arg(package_version_id);

-- name: HasApprovalFrom :one
select exists (
  select 1 from approvals
  where package_version_id = sqlc.arg(package_version_id) and reviewer_id = sqlc.arg(reviewer_id)
) as has_approval;

-- name: GetPackageVisibility :one
select p.visibility
from packages p
join namespaces n on n.id = p.namespace_id
join organizations o on o.id = n.org_id
where o.slug = sqlc.arg(org)
  and n.slug = sqlc.arg(namespace)
  and p.name = sqlc.arg(package_name);
