CREATE TABLE "support_messages" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "user_id" bigint NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
  "sender" varchar(10) NOT NULL,
  "author_id" bigint REFERENCES "users" ("id") ON DELETE SET NULL,
  "body" text NOT NULL,
  "read_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "support_messages" ("user_id", "id");
