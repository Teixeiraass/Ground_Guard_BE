CREATE TABLE "push_tokens" (
    "id" bigserial PRIMARY KEY,
    "user_id" bigint NOT NULL,
    "token" text UNIQUE NOT NULL,
    "platform" varchar(10) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT (now()),
    "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "push_tokens" ("user_id");

ALTER TABLE "push_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
