CREATE TABLE "irrigation_preferences_history" (
  "id" bigserial PRIMARY KEY,
  "preference_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "old_data" jsonb,
  "new_data" jsonb,
  "changed_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "irrigation_preferences_history" ADD FOREIGN KEY ("preference_id") REFERENCES "irrigation_preferences" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "irrigation_preferences_history" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;