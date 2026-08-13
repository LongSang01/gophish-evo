
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Activity source type: email | client | page
ALTER TABLE campaigns ADD COLUMN source_type TEXT NOT NULL DEFAULT 'email';

-- Per-campaign dynamic field configuration for client/page type activities.
-- Format: {"fields":[...],"dedup_key":"mac"}
ALTER TABLE campaigns ADD COLUMN report_config_json TEXT NOT NULL DEFAULT '{}';

-- Per-campaign random salt used to derive the report authentication key.
-- Persisted in the database so keys survive server restarts.
ALTER TABLE campaigns ADD COLUMN report_salt TEXT NOT NULL DEFAULT '';

-- Table holding reported data from the client and fixed-page modules.
CREATE TABLE IF NOT EXISTS "reports_ext" (
    "id"          integer primary key autoincrement,
    "campaign_id" bigint NOT NULL,
    "source"      text NOT NULL,
    "data_json"   text NOT NULL DEFAULT '{}',
    "ip"          text,
    "user_agent"  text,
    "dedup_value" text,
    "created_at"  datetime
);

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_reports_ext_dedup" ON "reports_ext" ("campaign_id", "source", "dedup_value");

-- +goose Down
DROP TABLE IF EXISTS reports_ext;
-- Note: SQLite does not support DROP COLUMN directly, so rolling back the
-- campaigns columns requires recreating the table, which is out of scope.
