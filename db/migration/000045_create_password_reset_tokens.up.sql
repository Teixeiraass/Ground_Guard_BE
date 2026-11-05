CREATE TABLE "password_reset_tokens" (
    "id" bigserial PRIMARY KEY,
    "user_id" bigint NOT NULL,
    "token_hash" varchar NOT NULL,
    "expires_at" timestamptz NOT NULL,
    "used_at" timestamptz,
    "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "password_reset_tokens" ("user_id");
CREATE INDEX ON "password_reset_tokens" ("token_hash");

ALTER TABLE "password_reset_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
