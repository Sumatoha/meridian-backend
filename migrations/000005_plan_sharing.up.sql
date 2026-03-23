-- Add share token to content_plans for public sharing (Business tier)
ALTER TABLE content_plans ADD COLUMN share_token TEXT UNIQUE;

CREATE INDEX idx_plans_share_token ON content_plans(share_token) WHERE share_token IS NOT NULL;
