-- Drop foreign key constraints first
ALTER TABLE "public"."servers" DROP CONSTRAINT "fk_servers_private_key_id_private_keys_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT "fk_team_invites_Invited_by_users_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT "fk_team_invites_Invited_to_users_id";
ALTER TABLE "public"."teams" DROP CONSTRAINT "fk_teams_owner_id_users_id";
ALTER TABLE "public"."team_users" DROP CONSTRAINT "fk_team_users_user_id_users_id";
ALTER TABLE "public"."team_users" DROP CONSTRAINT "fk_team_users_team_id_teams_id";
ALTER TABLE "public"."team_invites" DROP CONSTRAINT "fk_team_invites_team_id_teams_id";
ALTER TABLE "public"."servers" DROP CONSTRAINT "fk_servers_team_id_teams_id";
ALTER TABLE "public"."refresh_tokens" DROP CONSTRAINT "fk_refresh_tokens_user_id_users_id";
ALTER TABLE "public"."private_keys" DROP CONSTRAINT "fk_private_keys_team_id_teams_id";
ALTER TABLE "public"."oauth_tokens" DROP CONSTRAINT "fk_oauth_tokens_account_id_accounts_id";
ALTER TABLE "public"."accounts" DROP CONSTRAINT "fk_accounts_user_id_users_id";

-- Drop indexes
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
DROP TABLE IF EXISTS "public"."oauth_tokens";
DROP TABLE IF EXISTS "public"."servers";
DROP TABLE IF EXISTS "public"."private_keys";
DROP TABLE IF EXISTS "public"."team_invites";
DROP TABLE IF EXISTS "public"."team_users";
DROP TABLE IF EXISTS "public"."teams";
DROP TABLE IF EXISTS "public"."accounts";
DROP TABLE IF EXISTS "public"."refresh_tokens";
DROP TABLE IF EXISTS "public"."users";