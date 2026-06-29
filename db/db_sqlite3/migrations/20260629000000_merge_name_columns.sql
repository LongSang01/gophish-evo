
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- SQLite does not support ALTER TABLE DROP COLUMN before 3.35.0.
-- We need to recreate the tables without the old columns.

-- Create temporary table for targets
CREATE TABLE IF NOT EXISTS "targets_temp" (
    "id" integer primary key autoincrement,
    "full_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255)
);

-- Copy data from targets to targets_temp
INSERT INTO "targets_temp" ("id", "full_name", "email", "position")
SELECT "id", TRIM(COALESCE("first_name", '') || ' ' || COALESCE("last_name", '')), "email", "position"
FROM "targets";

-- Drop old table and rename new one
DROP TABLE "targets";
ALTER TABLE "targets_temp" RENAME TO "targets";

-- Create temporary table for results
CREATE TABLE IF NOT EXISTS "results_temp" (
    "id" integer primary key autoincrement,
    "campaign_id" bigint,
    "user_id" bigint,
    "r_id" varchar(255),
    "email" varchar(255),
    "full_name" varchar(255),
    "status" varchar(255) NOT NULL,
    "ip" varchar(255),
    "latitude" real,
    "longitude" real
);

-- Copy data from results to results_temp
INSERT INTO "results_temp" ("id", "campaign_id", "user_id", "r_id", "email", "full_name", "status", "ip", "latitude", "longitude")
SELECT "id", "campaign_id", "user_id", "r_id", "email", TRIM(COALESCE("first_name", '') || ' ' || COALESCE("last_name", '')), "status", "ip", "latitude", "longitude"
FROM "results";

-- Drop old table and rename new one
DROP TABLE "results";
ALTER TABLE "results_temp" RENAME TO "results";

-- Create temporary table for email_requests
CREATE TABLE IF NOT EXISTS "email_requests_temp" (
    "id" integer primary key autoincrement,
    "user_id" integer,
    "template_id" integer,
    "page_id" integer,
    "full_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255),
    "url" varchar(255),
    "r_id" varchar(255),
    "from_address" varchar(255)
);

-- Copy data from email_requests to email_requests_temp
INSERT INTO "email_requests_temp" ("id", "user_id", "template_id", "page_id", "full_name", "email", "position", "url", "r_id", "from_address")
SELECT "id", "user_id", "template_id", "page_id", TRIM(COALESCE("first_name", '') || ' ' || COALESCE("last_name", '')), "email", "position", "url", "r_id", "from_address"
FROM "email_requests";

-- Drop old table and rename new one
DROP TABLE "email_requests";
ALTER TABLE "email_requests_temp" RENAME TO "email_requests";

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- Create temporary table for targets with original columns
CREATE TABLE IF NOT EXISTS "targets_temp" (
    "id" integer primary key autoincrement,
    "first_name" varchar(255),
    "last_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255)
);

-- Copy data from targets to targets_temp
INSERT INTO "targets_temp" ("id", "first_name", "last_name", "email", "position")
SELECT "id", 
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", 1, INSTR("full_name", ' ') - 1) ELSE "full_name" END,
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", INSTR("full_name", ' ') + 1) ELSE '' END,
    "email", "position"
FROM "targets";

-- Drop old table and rename new one
DROP TABLE "targets";
ALTER TABLE "targets_temp" RENAME TO "targets";

-- Create temporary table for results with original columns
CREATE TABLE IF NOT EXISTS "results_temp" (
    "id" integer primary key autoincrement,
    "campaign_id" bigint,
    "user_id" bigint,
    "r_id" varchar(255),
    "email" varchar(255),
    "first_name" varchar(255),
    "last_name" varchar(255),
    "status" varchar(255) NOT NULL,
    "ip" varchar(255),
    "latitude" real,
    "longitude" real
);

-- Copy data from results to results_temp
INSERT INTO "results_temp" ("id", "campaign_id", "user_id", "r_id", "email", "first_name", "last_name", "status", "ip", "latitude", "longitude")
SELECT "id", "campaign_id", "user_id", "r_id", "email",
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", 1, INSTR("full_name", ' ') - 1) ELSE "full_name" END,
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", INSTR("full_name", ' ') + 1) ELSE '' END,
    "status", "ip", "latitude", "longitude"
FROM "results";

-- Drop old table and rename new one
DROP TABLE "results";
ALTER TABLE "results_temp" RENAME TO "results";

-- Create temporary table for email_requests with original columns
CREATE TABLE IF NOT EXISTS "email_requests_temp" (
    "id" integer primary key autoincrement,
    "user_id" integer,
    "template_id" integer,
    "page_id" integer,
    "first_name" varchar(255),
    "last_name" varchar(255),
    "email" varchar(255),
    "position" varchar(255),
    "url" varchar(255),
    "r_id" varchar(255),
    "from_address" varchar(255)
);

-- Copy data from email_requests to email_requests_temp
INSERT INTO "email_requests_temp" ("id", "user_id", "template_id", "page_id", "first_name", "last_name", "email", "position", "url", "r_id", "from_address")
SELECT "id", "user_id", "template_id", "page_id",
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", 1, INSTR("full_name", ' ') - 1) ELSE "full_name" END,
    CASE WHEN INSTR("full_name", ' ') > 0 THEN SUBSTR("full_name", INSTR("full_name", ' ') + 1) ELSE '' END,
    "email", "position", "url", "r_id", "from_address"
FROM "email_requests";

-- Drop old table and rename new one
DROP TABLE "email_requests";
ALTER TABLE "email_requests_temp" RENAME TO "email_requests";
