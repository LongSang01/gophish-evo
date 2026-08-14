
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Table for aggregated page click statistics, flushed from memory periodically.
CREATE TABLE IF NOT EXISTS "page_click_stats" (
    "id"           integer primary key autoincrement,
    "campaign_id"  bigint NOT NULL,
    "vid"          text NOT NULL DEFAULT '',
    "click_count"  bigint NOT NULL DEFAULT 0,
    "first_seen_at" datetime,
    "last_seen_at"  datetime
);

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_page_click_stats_campaign_vid"
    ON "page_click_stats" ("campaign_id", "vid");

-- +goose Down
DROP TABLE IF EXISTS page_click_stats;
