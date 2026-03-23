DROP INDEX IF EXISTS idx_plans_share_token;
ALTER TABLE content_plans DROP COLUMN IF EXISTS share_token;
