CREATE TABLE "device_groups" (
    "id" bigserial PRIMARY KEY,
    "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
    "user_id" bigint NOT NULL,
    "name" varchar(100) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT (now()),
    "updated_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "device_groups" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "devices" ADD COLUMN "group_id" bigint;
ALTER TABLE "devices" ADD FOREIGN KEY ("group_id") REFERENCES "device_groups" ("id") ON DELETE SET NULL DEFERRABLE INITIALLY IMMEDIATE;
CREATE INDEX ON "devices" ("group_id");

ALTER TABLE "irrigation_schedules" ADD COLUMN "group_id" bigint;
ALTER TABLE "irrigation_schedules" ADD FOREIGN KEY ("group_id") REFERENCES "device_groups" ("id") ON DELETE SET NULL DEFERRABLE INITIALLY IMMEDIATE;
