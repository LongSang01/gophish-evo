
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Create the campaign_smtps join table
CREATE TABLE IF NOT EXISTS `campaign_smtps` (
    `id`          bigint AUTO_INCREMENT PRIMARY KEY,
    `campaign_id` bigint NOT NULL,
    `smtp_id`     bigint NOT NULL,
    `position`    int NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Backfill existing single-SMTP campaigns
INSERT INTO campaign_smtps (campaign_id, smtp_id, position)
SELECT id, smtp_id, 0 FROM campaigns WHERE smtp_id IS NOT NULL AND smtp_id != 0;

-- Add smtp_id column to results table
ALTER TABLE results ADD COLUMN smtp_id bigint DEFAULT 0;

-- Backfill existing results
UPDATE results r
    INNER JOIN campaigns c ON c.id = r.campaign_id
SET r.smtp_id = c.smtp_id
WHERE r.smtp_id = 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS campaign_smtps;
ALTER TABLE results DROP COLUMN smtp_id;
