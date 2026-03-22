-- Drop all RLS policies
DROP POLICY IF EXISTS "users_own_row" ON users;
DROP POLICY IF EXISTS "users_own_accounts" ON instagram_accounts;
DROP POLICY IF EXISTS "users_own_settings" ON brand_settings;
DROP POLICY IF EXISTS "users_own_dna" ON brand_dna;
DROP POLICY IF EXISTS "users_own_plans" ON content_plans;
DROP POLICY IF EXISTS "users_own_slots" ON content_slots;
DROP POLICY IF EXISTS "users_own_scraped" ON scraped_posts;
DROP POLICY IF EXISTS "users_own_payments" ON payments;

-- Disable RLS
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE instagram_accounts DISABLE ROW LEVEL SECURITY;
ALTER TABLE brand_settings DISABLE ROW LEVEL SECURITY;
ALTER TABLE brand_dna DISABLE ROW LEVEL SECURITY;
ALTER TABLE content_plans DISABLE ROW LEVEL SECURITY;
ALTER TABLE content_slots DISABLE ROW LEVEL SECURITY;
ALTER TABLE scraped_posts DISABLE ROW LEVEL SECURITY;
ALTER TABLE payments DISABLE ROW LEVEL SECURITY;
