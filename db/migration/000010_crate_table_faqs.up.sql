CREATE TABLE "faqs" (
  "id" bigserial PRIMARY KEY,
  "uuid" UUID UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "question" varchar(255) NOT NULL,
  "answer" text NOT NULL,
  "category" varchar(100),
  "published" boolean NOT NULL DEFAULT true,
  "order_number" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE UNIQUE INDEX ON "faqs" ("uuid");

CREATE INDEX ON "faqs" ("category");
