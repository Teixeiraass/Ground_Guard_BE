CREATE TABLE "notifications" (
    "id" bigserial PRIMARY KEY,
    "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
    "user_id" bigint NOT NULL,
    "type" varchar(40) NOT NULL,
    "source" varchar(20) NOT NULL DEFAULT 'SYSTEM',
    "title" varchar(140) NOT NULL,
    "body" text NOT NULL,
    "read_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "notifications" ("user_id", "read_at");

ALTER TABLE "notifications" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
