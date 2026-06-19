CREATE TABLE "tutorials" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "title" varchar(200) NOT NULL,
  "description" text,
  "content" text NOT NULL,
  "image_url" varchar(255),
  "video_url" varchar(255),
  "category" varchar(100),
  "published" boolean NOT NULL DEFAULT true,
  "order_number" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "tutorials" ("uuid");

CREATE INDEX ON "tutorials" ("category");