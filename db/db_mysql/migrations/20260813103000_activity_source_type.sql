
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE campaigns ADD COLUMN source_type VARCHAR(20) NOT NULL DEFAULT 'email';
ALTER TABLE campaigns ADD COLUMN report_config_json TEXT;
ALTER TABLE campaigns ADD COLUMN report_salt VARCHAR(64) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS reports_ext (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id BIGINT NOT NULL,
    vid         VARCHAR(64) DEFAULT '',
    data_json   TEXT NOT NULL,
    ip          VARCHAR(64),
    user_agent  VARCHAR(512),
    dedup_value VARCHAR(255) DEFAULT NULL,
    created_at  DATETIME,
    INDEX idx_reports_ext_campaign_vid (campaign_id, vid),
    -- Exact-match dedup for client campaigns. NULL rows (page submissions /
    -- submissions without the dedup key) are never deduplicated by this index.
    UNIQUE KEY uniq_reports_ext_campaign_dedup (campaign_id, dedup_value)
) ENGINE=InnoDB;

-- Hot path: campaign-scoped result/stat queries filter on campaign_id.
CREATE INDEX idx_results_campaign_id ON results (campaign_id);

-- +goose Down
DROP TABLE IF EXISTS reports_ext;
DROP INDEX idx_results_campaign_id ON results;
ALTER TABLE campaigns DROP COLUMN source_type;
ALTER TABLE campaigns DROP COLUMN report_config_json;
ALTER TABLE campaigns DROP COLUMN report_salt;
