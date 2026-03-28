ALTER TABLE content_plans DROP COLUMN IF EXISTS error_message;

ALTER TABLE content_plans DROP CONSTRAINT IF EXISTS content_plans_status_check;
ALTER TABLE content_plans ADD CONSTRAINT content_plans_status_check
    CHECK (status IN ('draft', 'active', 'completed', 'archived'));

-- Clean up any plans with new statuses
UPDATE content_plans SET status = 'draft' WHERE status IN ('generating', 'failed');
