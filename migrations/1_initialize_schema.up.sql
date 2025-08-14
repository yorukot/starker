CREATE SCHEMA IF NOT EXISTS "public";

CREATE TABLE "public"."users" (
    "id" character varying(27) NOT NULL,
    "password_hash" text,
    "avatar" text,
    "display_name" text NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."refresh_tokens" (
    "id" character varying(27) NOT NULL,
    "user_id" character varying(27) NOT NULL,
    "token" text NOT NULL UNIQUE,
    "user_agent" text,
    "ip" inet,
    "used_at" timestamp,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE UNIQUE INDEX "refresh_tokens_refresh_tokens_token_key" ON "public"."refresh_tokens" ("token");
CREATE INDEX "refresh_tokens_idx_refresh_tokens_token" ON "public"."refresh_tokens" ("token");
CREATE INDEX "refresh_tokens_idx_refresh_tokens_created_at" ON "public"."refresh_tokens" ("created_at");
CREATE INDEX "refresh_tokens_idx_refresh_tokens_user_id" ON "public"."refresh_tokens" ("user_id");

CREATE TABLE "public"."team_invites" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "Invited_by" character varying(27) NOT NULL,
    "Invited_to" character varying(27) NOT NULL,
    "updated_at" timestamp NOT NULL,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."projects" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "updated_at" timestamp NOT NULL,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."team_users" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "user_id" character varying(27) NOT NULL,
    "updated_at" timestamp NOT NULL,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."service_environment_variables" (
    "id" character varying(27) NOT NULL,
    "service_id" character varying(27) NOT NULL,
    "key" text NOT NULL,
    "value" text NOT NULL,
    "is_secret" boolean DEFAULT FALSE,
    "created_at" timestamp NOT NULL,
    "updated_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE INDEX "service_environment_variables_idx_service_id" ON "public"."service_environment_variables" ("service_id");
CREATE UNIQUE INDEX "service_environment_variables_service_key_unique" ON "public"."service_environment_variables" ("service_id", "key");

CREATE TABLE "public"."servers" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "ip" text NOT NULL,
    "port" text NOT NULL,
    "user" text NOT NULL,
    "private_key_id" character varying(27) NOT NULL,
    "updated_at" timestamp NOT NULL,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."service_port_mappings" (
    "id" character varying(27) NOT NULL,
    "service_id" character varying(27) NOT NULL,
    "host_port" integer NOT NULL,
    "container_port" integer NOT NULL,
    "protocol" varchar(10) DEFAULT 'tcp',
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE INDEX "service_port_mappings_service_port_mappings_idx_service_id" ON "public"."service_port_mappings" ("service_id");
CREATE UNIQUE INDEX "service_port_mappings_service_host_port_unique" ON "public"."service_port_mappings" ("service_id", "host_port");

CREATE TABLE "public"."services" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "server_id" character varying(27) NOT NULL,
    "project_id" character varying(27) NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "type" character varying(20) NOT NULL,
    "status" character varying(20) DEFAULT 'stopped',
    "container_id" text,
    "last_deployed_at" timestamp,
    "created_at" timestamp NOT NULL,
    "updated_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE INDEX "services_services_idx_team_id" ON "public"."services" ("team_id");
CREATE INDEX "services_services_idx_server_id" ON "public"."services" ("server_id");
CREATE INDEX "services_services_idx_status" ON "public"."services" ("status");
CREATE INDEX "services_services_idx_type" ON "public"."services" ("type");

CREATE TABLE "public"."private_keys" (
    "id" character varying(27) NOT NULL,
    "team_id" character varying(27) NOT NULL,
    "name" text NOT NULL,
    "description" text,
    "private_key" text NOT NULL,
    "fingerprint" text NOT NULL,
    "created_at" timestamp NOT NULL,
    "updated_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."accounts" (
    "id" character varying(27) NOT NULL,
    "provider" character varying(50) NOT NULL,
    "provider_user_id" character varying(255) NOT NULL,
    "user_id" character varying(27) NOT NULL,
    "email" character varying(255) NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE UNIQUE INDEX "accounts_accounts_provider_provider_user_id_key" ON "public"."accounts" ("provider", "provider_user_id");
CREATE INDEX "accounts_idx_accounts_provider" ON "public"."accounts" ("provider");
CREATE UNIQUE INDEX "accounts_accounts_provider_email_key" ON "public"."accounts" ("provider", "email");
CREATE INDEX "accounts_idx_accounts_email" ON "public"."accounts" ("email");
CREATE INDEX "accounts_idx_accounts_user_id" ON "public"."accounts" ("user_id");

CREATE TABLE "public"."service_compose_configs" (
    "id" character varying(27) NOT NULL,
    "service_id" character varying(27) NOT NULL,
    "compose_file" text NOT NULL,
    "compose_file_path" text,
    "created_at" timestamp NOT NULL,
    "updated_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);
-- Indexes
CREATE UNIQUE INDEX "service_compose_configs_service_id_unique" ON "public"."service_compose_configs" ("service_id");

CREATE TABLE "public"."oauth_tokens" (
    "account_id" character varying(27) NOT NULL,
    "access_token" text NOT NULL,
    "refresh_token" text,
    "expiry" timestamp with time zone NOT NULL,
    "token_type" character varying(50) NOT NULL,
    "provider" character varying(50) NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    PRIMARY KEY ("account_id")
);
-- Indexes
CREATE INDEX "oauth_tokens_idx_oauth_tokens_provider" ON "public"."oauth_tokens" ("provider");

CREATE TABLE "public"."teams" (
    "id" character varying(27) NOT NULL,
    "owner_id" character varying(27) NOT NULL,
    "name" text NOT NULL,
    "updated_at" timestamp NOT NULL,
    "created_at" timestamp NOT NULL,
    PRIMARY KEY ("id")
);

-- Foreign key constraints
-- Schema: public
ALTER TABLE "public"."accounts" ADD CONSTRAINT "fk_accounts_user_id_users_id" FOREIGN KEY("user_id") REFERENCES "public"."users"("id");
ALTER TABLE "public"."service_compose_configs" ADD CONSTRAINT "fk_service_compose_configs_service_id_services_id" FOREIGN KEY("service_id") REFERENCES "public"."services"("id");
ALTER TABLE "public"."service_environment_variables" ADD CONSTRAINT "fk_service_environment_variables_service_id_services_id" FOREIGN KEY("service_id") REFERENCES "public"."services"("id");
ALTER TABLE "public"."service_port_mappings" ADD CONSTRAINT "fk_service_port_mappings_service_id_services_id" FOREIGN KEY("service_id") REFERENCES "public"."services"("id");
ALTER TABLE "public"."oauth_tokens" ADD CONSTRAINT "fk_oauth_tokens_account_id_accounts_id" FOREIGN KEY("account_id") REFERENCES "public"."accounts"("id");
ALTER TABLE "public"."private_keys" ADD CONSTRAINT "fk_private_keys_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."projects" ADD CONSTRAINT "fk_projects_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."refresh_tokens" ADD CONSTRAINT "fk_refresh_tokens_user_id_users_id" FOREIGN KEY("user_id") REFERENCES "public"."users"("id");
ALTER TABLE "public"."servers" ADD CONSTRAINT "fk_servers_private_key_id_private_keys_id" FOREIGN KEY("private_key_id") REFERENCES "public"."private_keys"("id");
ALTER TABLE "public"."servers" ADD CONSTRAINT "fk_servers_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."services" ADD CONSTRAINT "fk_services_project_id_projects_id" FOREIGN KEY("project_id") REFERENCES "public"."projects"("id");
ALTER TABLE "public"."services" ADD CONSTRAINT "fk_services_server_id_servers_id" FOREIGN KEY("server_id") REFERENCES "public"."servers"("id");
ALTER TABLE "public"."services" ADD CONSTRAINT "fk_services_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."team_invites" ADD CONSTRAINT "fk_team_invites_Invited_by_users_id" FOREIGN KEY("Invited_by") REFERENCES "public"."users"("id");
ALTER TABLE "public"."team_invites" ADD CONSTRAINT "fk_team_invites_Invited_to_users_id" FOREIGN KEY("Invited_to") REFERENCES "public"."users"("id");
ALTER TABLE "public"."team_invites" ADD CONSTRAINT "fk_team_invites_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."team_users" ADD CONSTRAINT "fk_team_users_team_id_teams_id" FOREIGN KEY("team_id") REFERENCES "public"."teams"("id");
ALTER TABLE "public"."team_users" ADD CONSTRAINT "fk_team_users_user_id_users_id" FOREIGN KEY("user_id") REFERENCES "public"."users"("id");
ALTER TABLE "public"."teams" ADD CONSTRAINT "fk_teams_owner_id_users_id" FOREIGN KEY("owner_id") REFERENCES "public"."users"("id");
