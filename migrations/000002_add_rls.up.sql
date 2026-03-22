-- Enable Row Level Security on all tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE instagram_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE brand_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE brand_dna ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_slots ENABLE ROW LEVEL SECURITY;
ALTER TABLE scraped_posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

-- Policy: users can only see their own row
CREATE POLICY "users_own_row" ON users
    FOR ALL USING (supabase_user_id = auth.uid());

-- Policy: users can only access their own Instagram accounts
CREATE POLICY "users_own_accounts" ON instagram_accounts
    FOR ALL USING (user_id IN (
        SELECT id FROM users WHERE supabase_user_id = auth.uid()
    ));

-- Policy: brand_settings scoped to user's accounts
CREATE POLICY "users_own_settings" ON brand_settings
    FOR ALL USING (instagram_account_id IN (
        SELECT id FROM instagram_accounts WHERE user_id IN (
            SELECT id FROM users WHERE supabase_user_id = auth.uid()
        )
    ));

-- Policy: brand_dna scoped to user's accounts
CREATE POLICY "users_own_dna" ON brand_dna
    FOR ALL USING (instagram_account_id IN (
        SELECT id FROM instagram_accounts WHERE user_id IN (
            SELECT id FROM users WHERE supabase_user_id = auth.uid()
        )
    ));

-- Policy: content_plans scoped to user's accounts
CREATE POLICY "users_own_plans" ON content_plans
    FOR ALL USING (instagram_account_id IN (
        SELECT id FROM instagram_accounts WHERE user_id IN (
            SELECT id FROM users WHERE supabase_user_id = auth.uid()
        )
    ));

-- Policy: content_slots scoped through plans → accounts → users
CREATE POLICY "users_own_slots" ON content_slots
    FOR ALL USING (plan_id IN (
        SELECT id FROM content_plans WHERE instagram_account_id IN (
            SELECT id FROM instagram_accounts WHERE user_id IN (
                SELECT id FROM users WHERE supabase_user_id = auth.uid()
            )
        )
    ));

-- Policy: scraped_posts scoped to user's accounts
CREATE POLICY "users_own_scraped" ON scraped_posts
    FOR ALL USING (instagram_account_id IN (
        SELECT id FROM instagram_accounts WHERE user_id IN (
            SELECT id FROM users WHERE supabase_user_id = auth.uid()
        )
    ));

-- Policy: payments scoped to user
CREATE POLICY "users_own_payments" ON payments
    FOR ALL USING (user_id IN (
        SELECT id FROM users WHERE supabase_user_id = auth.uid()
    ));

-- IMPORTANT: Allow the service role (backend) to bypass RLS
-- The backend connects as the service_role, which bypasses RLS by default in Supabase.
-- These policies only restrict direct client access (e.g., from Supabase JS SDK).
