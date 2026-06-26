
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Create the campaign_smtps join table
CREATE TABLE IF NOT EXISTS "campaign_smtps" (
    "id"          integer primary key autoincrement,
    "campaign_id" bigint NOT NULL,
    "smtp_id"     bigint NOT NULL,
    "position"    integer NOT NULL DEFAULT 0
);

-- Backfill: for every existing campaign that has a single smtp_id,
-- create one campaign_smtps row at position 0 so the new code can
-- read old campaigns without special-casing.
INSERT INTO campaign_smtps (campaign_id, smtp_id, position)
SELECT id, smtp_id, 0 FROM campaigns WHERE smtp_id IS NOT NULL AND smtp_id != 0;

-- Add smtp_id column to results table to record which SMTP was
-- assigned to each recipient.
ALTER TABLE results ADD COLUMN smtp_id bigint DEFAULT 0;

-- Backfill existing results with the campaign's single SMTP.
UPDATE results SET smtp_id = (
    SELECT c.smtp_id FROM campaigns c WHERE c.id = results.campaign_id
) WHERE smtp_id = 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS campaign_smtps;
-- Note: SQLite does not support DROP COLUMN directly.
-- To fully roll back the results.smtp_id column you would need to
-- recreate the table without it, which is out of scope for this
-- automated down-migration.
