
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE campaigns ADD COLUMN source_type VARCHAR(20) NOT NULL DEFAULT 'email';
ALTER TABLE campaigns ADD COLUMN report_config_json TEXT;
ALTER TABLE campaigns ADD COLUMN report_salt VARCHAR(64) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS reports_ext (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id BIGINT NOT NULL,
    source      VARCHAR(20) NOT NULL,
    data_json   TEXT NOT NULL,
    ip          VARCHAR(64),
    user_agent  VARCHAR(512),
    dedup_value VARCHAR(255),
    created_at  DATETIME,
    UNIQUE KEY uniq_reports_ext_dedup (campaign_id, source, dedup_value)
) ENGINE=InnoDB;

-- +goose Down
DROP TABLE IF EXISTS reports_ext;
ALTER TABLE campaigns DROP COLUMN source_type;
ALTER TABLE campaigns DROP COLUMN report_config_json;
ALTER TABLE campaigns DROP COLUMN report_salt;
