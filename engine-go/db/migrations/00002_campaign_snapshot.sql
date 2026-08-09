-- Snapshot-per-campaign (ALE-33): joining a campaign clones the character into a
-- campaign-scoped copy so edits don't leak across mesas. A template has both
-- columns NULL; a copy carries its origin (sourceCharacterId) + owning campaign.
-- +goose Up
ALTER TABLE characters ADD COLUMN sourceCharacterId INTEGER REFERENCES characters(id) ON DELETE SET NULL;
ALTER TABLE characters ADD COLUMN campaignId INTEGER REFERENCES campaigns(id) ON DELETE CASCADE;
CREATE INDEX idx_characters_campaignId ON characters(campaignId);

-- +goose Down
DROP INDEX idx_characters_campaignId;
ALTER TABLE characters DROP COLUMN campaignId;
ALTER TABLE characters DROP COLUMN sourceCharacterId;
