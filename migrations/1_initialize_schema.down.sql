-- Drop foreign key constraints first
ALTER TABLE "public"."service_volume" DROP CONSTRAINT IF EXISTS "fk_service_volume_service_id_services_id";
ALTER TABLE "public"."service_images" DROP CONSTRAINT IF EXISTS "fk_service_images_service_id_services_id";
ALTER TABLE "public"."service_network" DROP CONSTRAINT IF EXISTS "fk_service_network_service_id_services_id";
ALTER TABLE "public"."service_containers" DROP CONSTRAINT IF EXISTS "fk_service_containers_service_id_services_id";
ALTER TABLE "public"."service_source_git" DROP CONSTRAINT IF EXISTS "fk_service_source_git_service_id_services_id";
ALTER TABLE "public"."teams" DROP CONSTRAINT IF EXISTS "fk_teams_owner_id_users_id";
ALTER TABLE "public"."team_users" DROP CONSTRAINT IF EXISTS "fk_team_users_user_id_users_id";
ALTER TABLE "public"."team_users" DROP CONSTRAINT IF EXISTS "fk_team_users_team_id_teams_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT IF EXISTS "fk_team_invites_team_id_teams_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT IF EXISTS "fk_team_invites_Invited_to_users_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT IF EXISTS "fk_team_invites_Invited_by_users_id";
ALTER TABLE "public"."services" DROP CONSTRAINT IF EXISTS "fk_services_team_id_teams_id";
ALTER TABLE "public"."services" DROP CONSTRAINT IF EXISTS "fk_services_server_id_servers_id";
ALTER TABLE "public"."services" DROP CONSTRAINT IF EXISTS "fk_services_project_id_projects_id";
ALTER TABLE "public"."servers" DROP CONSTRAINT IF EXISTS "fk_servers_team_id_teams_id";
ALTER TABLE "public"."servers" DROP CONSTRAINT IF EXISTS "fk_servers_private_key_id_private_keys_id";
ALTER TABLE "public"."refresh_tokens" DROP CONSTRAINT IF EXISTS "fk_refresh_tokens_user_id_users_id";
ALTER TABLE "public"."projects" DROP CONSTRAINT IF EXISTS "fk_projects_team_id_teams_id";
ALTER TABLE "public"."private_keys" DROP CONSTRAINT IF EXISTS "fk_private_keys_team_id_teams_id";
ALTER TABLE "public"."oauth_tokens" DROP CONSTRAINT IF EXISTS "fk_oauth_tokens_account_id_accounts_id";
ALTER TABLE "public"."service_compose_configs" DROP CONSTRAINT IF EXISTS "fk_service_compose_configs_service_id_services_id";
ALTER TABLE "public"."accounts" DROP CONSTRAINT IF EXISTS "fk_accounts_user_id_users_id";

-- Drop indexes
DROP INDEX IF EXISTS "service_compose_configs_service_id_unique";
DROP INDEX IF EXISTS "services_services_idx_type";
DROP INDEX IF EXISTS "services_services_idx_status";
DROP INDEX IF EXISTS "services_services_idx_server_id";
DROP INDEX IF EXISTS "services_services_idx_team_id";
DROP INDEX IF EXISTS "oauth_tokens_idx_oauth_tokens_provider";
DROP INDEX IF EXISTS "accounts_idx_accounts_user_id";
DROP INDEX IF EXISTS "accounts_idx_accounts_email";
DROP INDEX IF EXISTS "accounts_accounts_provider_email_key";
DROP INDEX IF EXISTS "accounts_idx_accounts_provider";
DROP INDEX IF EXISTS "accounts_accounts_provider_provider_user_id_key";
DROP INDEX IF EXISTS "refresh_tokens_idx_refresh_tokens_user_id";
DROP INDEX IF EXISTS "refresh_tokens_idx_refresh_tokens_created_at";
DROP INDEX IF EXISTS "refresh_tokens_idx_refresh_tokens_token";
DROP INDEX IF EXISTS "refresh_tokens_refresh_tokens_token_key";

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS "public"."service_volume";
DROP TABLE IF EXISTS "public"."service_network";
DROP TABLE IF EXISTS "public"."service_images";
DROP TABLE IF EXISTS "public"."service_containers";
DROP TABLE IF EXISTS "public"."service_source_git";
DROP TABLE IF EXISTS "public"."service_compose_configs";
DROP TABLE IF EXISTS "public"."services";
DROP TABLE IF EXISTS "public"."projects";
DROP TABLE IF EXISTS "public"."oauth_tokens";
DROP TABLE IF EXISTS "public"."servers";
DROP TABLE IF EXISTS "public"."private_keys";
DROP TABLE IF EXISTS "public"."team_invites";
DROP TABLE IF EXISTS "public"."team_users";
DROP TABLE IF EXISTS "public"."teams";
DROP TABLE IF EXISTS "public"."accounts";
DROP TABLE IF EXISTS "public"."refresh_tokens";
DROP TABLE IF EXISTS "public"."users";