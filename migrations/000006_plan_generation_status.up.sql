-- Add 'generating' and 'failed' to plan status constraint
-- and add error_message column for failed generations

ALTER TABLE content_plans DROP CONSTRAINT IF EXISTS content_plans_status_check;
ALTER TABLE content_plans ADD CONSTRAINT content_plans_status_check
    CHECK (status IN ('generating', 'draft', 'active', 'completed', 'archived', 'failed'));

ALTER TABLE content_plans ADD COLUMN IF NOT EXISTS error_message TEXT;
