insert into organizations (id, slug, display_name, description, visibility, created_at, updated_at)
values (
  '0198f006-0000-7000-8000-000000000001',
  'companyx',
  'Company X',
  'Sample organization for the Trove Slice 1 read path.',
  'public',
  '2026-05-22T00:00:00Z',
  '2026-05-22T00:00:00Z'
);

insert into namespaces (id, org_id, slug, display_name, description, visibility, created_at, updated_at)
values (
  '0198f006-0000-7000-8000-000000000002',
  '0198f006-0000-7000-8000-000000000001',
  'platform',
  'Platform',
  'Shared platform agent artifacts.',
  'public',
  '2026-05-22T00:00:00Z',
  '2026-05-22T00:00:00Z'
);

insert into packages (id, org_id, namespace_id, name, display_name, description, visibility, lifecycle, created_at, updated_at)
values (
  '0198f006-0000-7000-8000-000000000003',
  '0198f006-0000-7000-8000-000000000001',
  '0198f006-0000-7000-8000-000000000002',
  'agent-backend',
  'Backend Agent Defaults',
  'Default agent instructions, skills, and commands for backend services.',
  'public',
  'active',
  '2026-05-22T00:00:00Z',
  '2026-05-22T00:00:00Z'
);

insert into package_versions (
  id,
  package_id,
  version,
  semver_major,
  semver_minor,
  semver_patch,
  lifecycle,
  visibility,
  manifest_json,
  changelog,
  digest,
  created_at,
  updated_at,
  published_at
)
values (
  '0198f006-0000-7000-8000-000000000004',
  '0198f006-0000-7000-8000-000000000003',
  '1.0.0',
  1,
  0,
  0,
  'draft',
  'public',
  '{"apiVersion":"trove.io/v1","kind":"AgentArtifactPackage","metadata":{"org":"companyx","namespace":"platform","name":"agent-backend","displayName":"Backend Agent Defaults","description":"Default agent instructions, skills, and commands for backend services.","labels":{"language":"golang","framework":"chi","maturity":"production"},"annotations":{"owner":"platform-engineering"}},"spec":{"version":"1.0.0","license":"internal","visibility":"public","lifecycle":"published","compatibility":{"tools":[{"name":"opencode","version":">=0.6.0 <2.0.0"}],"models":[{"family":"gpt","minContextWindow":128000}],"runtimes":["linux"]},"artifacts":[{"path":"AGENTS.md","type":"agent-instructions","required":true,"targetPath":"AGENTS.md"}],"dependencies":[],"updatePolicy":{"breakingChangeRequiresManualApproval":true},"maintainers":[{"team":"platform-engineering"}],"links":{"docs":"https://docs.company.com/agent-backend"}}}'::jsonb,
  'Initial seeded Slice 1 package.',
  'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050',
  '2026-05-22T00:00:00Z',
  '2026-05-22T00:00:00Z',
  null
);

insert into artifact_blobs (digest, content, size_bytes, created_at)
values (
  'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050',
  convert_to('# Backend Agent Defaults

Use Go, chi, pgx, and small reviewable changes.
', 'UTF8'),
  69,
  '2026-05-22T00:00:00Z'
);

insert into artifact_locations (digest, storage_driver, storage_uri, created_at)
values (
  'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050',
  'postgres',
  null,
  '2026-05-22T00:00:00Z'
);

insert into artifact_files (id, package_version_id, path, type, content_type, blob_digest, size_bytes, target_path, created_at)
values (
  '0198f006-0000-7000-8000-000000000005',
  '0198f006-0000-7000-8000-000000000004',
  'AGENTS.md',
  'agent-instructions',
  'text/markdown; charset=utf-8',
  'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050',
  69,
  'AGENTS.md',
  '2026-05-22T00:00:00Z'
);

update package_versions
set lifecycle = 'published',
    updated_at = '2026-05-22T00:00:00Z',
    published_at = '2026-05-22T00:00:00Z'
where id = '0198f006-0000-7000-8000-000000000004';

insert into audit_events (id, org_id, namespace_id, package_id, package_version_id, actor_service_account, action, metadata_json, created_at)
values (
  '0198f006-0000-7000-8000-000000000008',
  '0198f006-0000-7000-8000-000000000001',
  '0198f006-0000-7000-8000-000000000002',
  '0198f006-0000-7000-8000-000000000003',
  '0198f006-0000-7000-8000-000000000004',
  'seed',
  'package.seeded',
  '{"source":"migration"}'::jsonb,
  '2026-05-22T00:00:00Z'
);
