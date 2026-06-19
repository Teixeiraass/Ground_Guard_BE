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

CREATE UNIQUE INDEX ON "legal_documents" ("uuid");

CREATE INDEX ON "legal_documents" ("type");

CREATE INDEX ON "legal_documents" ("active");

CREATE UNIQUE INDEX ON "user_accepted_terms" ("uuid");

CREATE INDEX ON "user_accepted_terms" ("user_id");

CREATE INDEX ON "user_accepted_terms" ("legal_document_id");

ALTER TABLE "user_accepted_terms" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "user_accepted_terms" ADD FOREIGN KEY ("legal_document_id") REFERENCES "legal_documents" ("id") DEFERRABLE INITIALLY IMMEDIATE;