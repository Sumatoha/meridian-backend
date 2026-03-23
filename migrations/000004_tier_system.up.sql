-- Rename agency → business in users table
ALTER TABLE users DROP CONSTRAINT users_plan_check;
UPDATE users SET plan = 'business' WHERE plan = 'agency';
ALTER TABLE users ADD CONSTRAINT users_plan_check CHECK (plan IN ('free', 'pro', 'business'));

-- Rename agency → business in payments table
UPDATE payments SET plan = 'business' WHERE plan = 'agency';

-- Usage tracking table for monthly limits
CREATE TABLE usage_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN ('plan_generation')),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    used_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, action, period_start)
);

CREATE INDEX idx_usage_tracking_lookup ON usage_tracking(user_id, action, period_start);
