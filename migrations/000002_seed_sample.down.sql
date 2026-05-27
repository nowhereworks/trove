update package_versions
set lifecycle = 'draft',
    updated_at = '2026-05-22T00:00:00Z',
    published_at = null
where id = '0198f006-0000-7000-8000-000000000004';

delete from audit_events where id = '0198f006-0000-7000-8000-000000000008';
delete from channels where package_id = '0198f006-0000-7000-8000-000000000003';
delete from artifact_files where id = '0198f006-0000-7000-8000-000000000005';
delete from artifact_locations where digest = 'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050';
delete from artifact_blobs where digest = 'sha256:e794a7e6ed6c59d962237a4b1593fd158f10fa370beeebdd4bf9f6bf4734a050';
delete from package_versions where id = '0198f006-0000-7000-8000-000000000004';
delete from packages where id = '0198f006-0000-7000-8000-000000000003';
delete from namespaces where id = '0198f006-0000-7000-8000-000000000002';
delete from organizations where id = '0198f006-0000-7000-8000-000000000001';
