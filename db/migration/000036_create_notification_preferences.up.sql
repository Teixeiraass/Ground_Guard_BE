CREATE TABLE "notification_preferences" (
    "id" bigserial PRIMARY KEY,
    "user_id" bigint NOT NULL,
    "type" varchar(40) NOT NULL,
    "enabled" boolean NOT NULL DEFAULT true,
    "created_at" timestamptz NOT NULL DEFAULT (now()),
    "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "notification_preferences" ("user_id", "type");

ALTER TABLE "notification_preferences" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
