ALTER TABLE "users"
    ADD COLUMN "status" varchar(20) NOT NULL DEFAULT 'ACTIVE',
    ADD COLUMN "ban_reason" text,
    ADD COLUMN "banned_at" timestamptz,
    ADD COLUMN "is_admin" boolean NOT NULL DEFAULT false;
