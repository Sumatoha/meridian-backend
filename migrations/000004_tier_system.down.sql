-- Drop usage tracking
DROP INDEX IF EXISTS idx_usage_tracking_lookup;
DROP TABLE IF EXISTS usage_tracking;

-- Revert business → agency in users table
ALTER TABLE users DROP CONSTRAINT users_plan_check;
UPDATE users SET plan = 'agency' WHERE plan = 'business';
ALTER TABLE users ADD CONSTRAINT users_plan_check CHECK (plan IN ('free', 'pro', 'agency'));

-- Revert business → agency in payments table
UPDATE payments SET plan = 'agency' WHERE plan = 'business';
