-- Leads collected from the free public audit on the landing page.
-- Used for outreach, conversion tracking, and "X accounts analyzed" stats.
CREATE TABLE audit_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ig_username TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    locale TEXT,
    mock_score INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for dedup checks and stats queries
CREATE INDEX idx_audit_leads_username ON audit_leads (ig_username);
CREATE INDEX idx_audit_leads_created ON audit_leads (created_at DESC);
