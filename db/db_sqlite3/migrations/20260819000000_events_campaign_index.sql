-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Hot path: per-campaign email stats (getCampaignStats / GetDashboardStats)
-- count opened/clicked/submitted from the events table filtered on campaign_id.
CREATE INDEX IF NOT EXISTS "idx_events_campaign_id" ON "events" ("campaign_id");

-- +goose Down
DROP INDEX IF EXISTS "idx_events_campaign_id";