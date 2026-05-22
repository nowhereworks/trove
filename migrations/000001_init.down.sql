drop trigger if exists artifact_files_immutable_delete on artifact_files;
drop trigger if exists artifact_files_immutable_update on artifact_files;
drop trigger if exists artifact_files_immutable_insert on artifact_files;
drop function if exists prevent_published_artifact_change();

drop trigger if exists package_versions_immutable_update on package_versions;
drop function if exists prevent_published_version_update();

drop table if exists audit_events;
drop table if exists channels;
drop table if exists artifact_files;
drop table if exists artifact_locations;
drop table if exists artifact_blobs;
drop table if exists package_versions;
drop table if exists packages;
drop table if exists namespaces;
drop table if exists organizations;
