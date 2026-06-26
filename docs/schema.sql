-- SQL dump generated using DBML (dbml.dbdiagram.io)
-- Database: PostgreSQL
-- Generated at: 2026-06-26T18:44:38.243Z

CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE DEFAULT (gen_random_uuid()),
  "username" varchar NOT NULL,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "password_changed_at" timestampz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "profile_image" varchar(255)
);

CREATE TABLE "devices" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE DEFAULT (gen_random_uuid()),
  "device_uid" varchar(100) UNIQUE,
  "name" varchar(100),
  "firmware_version" varchar(50),
  "firmware_build" varchar(50),
  "last_update" timestamptz,
  "ip_address" inet,
  "qr_token" varchar(64) UNIQUE NOT NULL,
  "qr_code_file" varchar(255),
  "wifi_ssid" varchar(100),
  "last_seen" timestamptz,
  "status" varchar(20),
  "user_id" bigint
);

CREATE TABLE "irrigation_actions" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "device_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "started_at" timestamptz NOT NULL DEFAULT (now()),
  "finished_at" timestamptz,
  "duration_seconds" integer,
  "status" varchar(20) NOT NULL DEFAULT 'ATIVO',
  "trigger_type" varchar(20) NOT NULL,
  "water_volume_ml" integer,
  "error_message" text,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "irrigation_preferences" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "device_id" bigint UNIQUE NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "irrigation_mode" varchar(20) NOT NULL DEFAULT 'INTELIGENTE',
  "moisture_threshold" integer NOT NULL DEFAULT 30,
  "dry_time_minutes" integer NOT NULL DEFAULT 10,
  "irrigation_duration_seconds" integer NOT NULL DEFAULT 60,
  "max_irrigations_per_day" integer NOT NULL DEFAULT 10,
  "start_hour" time,
  "end_hour" time,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "irrigation_preferences_history" (
  "id" bigserial PRIMARY KEY,
  "preference_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "old_data" jsonb,
  "new_data" jsonb,
  "changed_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "help_contents" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "title" varchar(200) NOT NULL,
  "slug" varchar(200) UNIQUE NOT NULL,
  "category" varchar(50) NOT NULL,
  "content" text NOT NULL,
  "image_url" varchar(255),
  "published" boolean NOT NULL DEFAULT true,
  "order_number" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "faqs" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "question" varchar(255) NOT NULL,
  "answer" text NOT NULL,
  "category" varchar(100),
  "published" boolean NOT NULL DEFAULT true,
  "order_number" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "tutorials" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "title" varchar(200) NOT NULL,
  "description" text,
  "content" text NOT NULL,
  "image_url" varchar(255),
  "video_url" varchar(255),
  "category" varchar(100),
  "published" boolean NOT NULL DEFAULT true,
  "order_number" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "legal_documents" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "type" varchar(50) NOT NULL,
  "version" varchar(20) NOT NULL,
  "title" varchar(200) NOT NULL,
  "content" text NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "published_at" timestamptz NOT NULL DEFAULT (now()),
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "user_accepted_terms" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "user_id" bigint NOT NULL,
  "legal_document_id" bigint NOT NULL,
  "accepted_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "devices" ("user_id");

CREATE UNIQUE INDEX ON "devices" ("device_uid");

CREATE UNIQUE INDEX ON "devices" ("uuid");

CREATE INDEX ON "irrigation_actions" ("device_id");

CREATE INDEX ON "irrigation_actions" ("user_id");

CREATE INDEX ON "irrigation_actions" ("started_at");

CREATE UNIQUE INDEX ON "irrigation_actions" ("uuid");

CREATE UNIQUE INDEX ON "irrigation_preferences" ("device_id");

CREATE UNIQUE INDEX ON "irrigation_preferences" ("uuid");

CREATE UNIQUE INDEX ON "help_contents" ("uuid");

CREATE INDEX ON "help_contents" ("category");

CREATE UNIQUE INDEX ON "help_contents" ("slug");

CREATE UNIQUE INDEX ON "faqs" ("uuid");

CREATE INDEX ON "faqs" ("category");

CREATE UNIQUE INDEX ON "tutorials" ("uuid");

CREATE INDEX ON "tutorials" ("category");

CREATE UNIQUE INDEX ON "legal_documents" ("uuid");

CREATE INDEX ON "legal_documents" ("type");

CREATE INDEX ON "legal_documents" ("active");

CREATE UNIQUE INDEX ON "user_accepted_terms" ("uuid");

CREATE INDEX ON "user_accepted_terms" ("user_id");

CREATE INDEX ON "user_accepted_terms" ("legal_document_id");

ALTER TABLE "devices" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_actions" ADD FOREIGN KEY ("device_id") REFERENCES "devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_actions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_preferences" ADD FOREIGN KEY ("device_id") REFERENCES "devices" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_preferences_history" ADD FOREIGN KEY ("preference_id") REFERENCES "irrigation_preferences" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_preferences_history" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "user_accepted_terms" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "user_accepted_terms" ADD FOREIGN KEY ("legal_document_id") REFERENCES "legal_documents" ("id") DEFERRABLE INITIALLY IMMEDIATE;
