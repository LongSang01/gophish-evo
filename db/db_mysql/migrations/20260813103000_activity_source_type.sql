
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
    created_at  DATETIME,
    INDEX idx_reports_ext_campaign_vid (campaign_id, vid)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS reports_ext;
ALTER TABLE campaigns DROP COLUMN source_type;
ALTER TABLE campaigns DROP COLUMN report_config_json;
ALTER TABLE campaigns DROP COLUMN report_salt;
