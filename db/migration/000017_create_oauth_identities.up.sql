CREATE TABLE "oauth_identities" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "user_id" bigint NOT NULL,
  "provider" varchar(20) NOT NULL,
  "provider_subject" varchar(255) NOT NULL,
  "email" varchar(255) NOT NULL,
  "email_verified" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "oauth_identities" ("provider", "provider_subject");
CREATE INDEX ON "oauth_identities" ("user_id");
CREATE INDEX ON "oauth_identities" ("email");

ALTER TABLE "oauth_identities" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;