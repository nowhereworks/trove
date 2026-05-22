create table users (
  id uuid primary key,
  email text not null unique,
  display_name text not null,
  avatar_url text,
  oidc_issuer text,
  oidc_subject text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint users_email_check check (email <> '')
);

create table teams (
  id uuid primary key,
  org_id uuid not null references organizations(id),
  slug text not null,
  display_name text not null,
  description text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique(org_id, slug),
  constraint teams_slug_check check (slug ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$')
);

create table team_memberships (
  id uuid primary key,
  team_id uuid not null references teams(id),
  user_id uuid not null references users(id),
  role text not null default 'member',
  created_at timestamptz not null,
  unique(team_id, user_id),
  constraint team_memberships_role_check check (role in ('member', 'admin'))
);

create table package_maintainers (
  id uuid primary key,
  package_id uuid not null references packages(id),
  user_id uuid references users(id),
  team_id uuid references teams(id),
  created_at timestamptz not null,
  unique(package_id, user_id),
  unique(package_id, team_id),
  constraint package_maintainers_actor_check check (
    (user_id is not null and team_id is null) or
    (user_id is null and team_id is not null)
  )
);

create table reviews (
  id uuid primary key,
  package_version_id uuid not null references package_versions(id),
  reviewer_id uuid not null references users(id),
  status text not null,
  comment text,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  constraint reviews_status_check check (status in ('approved', 'changes_requested', 'submitted'))
);

create table review_comments (
  id uuid primary key,
  review_id uuid not null references reviews(id),
  author_id uuid not null references users(id),
  body text not null,
  created_at timestamptz not null
);

create table approvals (
  id uuid primary key,
  package_version_id uuid not null references package_versions(id),
  reviewer_id uuid not null references users(id),
  created_at timestamptz not null,
  unique(package_version_id, reviewer_id)
);

create table api_tokens (
  id uuid primary key,
  token_hash text not null unique,
  display_name text not null,
  actor_user_id uuid references users(id),
  actor_service_account text,
  scopes text[] not null,
  org_id uuid references organizations(id),
  namespace_id uuid references namespaces(id),
  package_id uuid references packages(id),
  created_at timestamptz not null,
  expires_at timestamptz,
  last_used_at timestamptz,
  revoked_at timestamptz,
  constraint api_tokens_actor_check check (
    actor_user_id is not null or actor_service_account is not null
  )
);

create index teams_org_slug_idx on teams(org_id, slug);
create index package_maintainers_package_idx on package_maintainers(package_id);
create index reviews_package_version_idx on reviews(package_version_id);
create index approvals_package_version_idx on approvals(package_version_id);
create index api_tokens_hash_idx on api_tokens(token_hash);
create index api_tokens_actor_user_idx on api_tokens(actor_user_id);
