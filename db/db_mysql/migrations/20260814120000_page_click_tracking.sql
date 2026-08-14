
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Add visitor ID (vid) column to reports_ext for page-type campaigns.
ALTER TABLE reports_ext ADD COLUMN vid VARCHAR(64) DEFAULT '';

-- Table for aggregated page click statistics, flushed from memory periodically.
CREATE TABLE IF NOT EXISTS page_click_stats (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id  BIGINT NOT NULL,
    ip           VARCHAR(64) NOT NULL DEFAULT '',
    vid          VARCHAR(64) NOT NULL DEFAULT '',
    click_count  BIGINT NOT NULL DEFAULT 0,
    first_seen_at DATETIME,
    last_seen_at  DATETIME,
    UNIQUE KEY uniq_page_click_stats_campaign_vid (campaign_id, vid)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS page_click_stats;
ALTER TABLE reports_ext DROP COLUMN vid;
