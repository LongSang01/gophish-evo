
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Table for aggregated page click statistics, flushed from memory periodically.
-- IP / user_agent reflect the most recently seen values so click-only
-- visitors (who never submit a form) still show this context in the report
-- timeline.
CREATE TABLE IF NOT EXISTS page_click_stats (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id  BIGINT NOT NULL,
    vid          VARCHAR(64) NOT NULL DEFAULT '',
    click_count  BIGINT NOT NULL DEFAULT 0,
    ip           VARCHAR(64) DEFAULT '',
    user_agent   VARCHAR(512) DEFAULT '',
    first_seen_at DATETIME,
    last_seen_at  DATETIME,
    UNIQUE KEY uniq_page_click_stats_campaign_vid (campaign_id, vid)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS page_click_stats;
