-- Users (synced from Supabase Auth, extended with our fields)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supabase_user_id UUID UNIQUE NOT NULL,
    email TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'pro', 'agency')),
    plan_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Instagram accounts connected by users
CREATE TABLE instagram_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ig_username TEXT NOT NULL,
    ig_user_id TEXT,
    access_token TEXT,
    token_expires_at TIMESTAMPTZ,
    profile_pic_url TEXT,
    followers_count INT,
    is_oauth_connected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Brand settings
CREATE TABLE brand_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instagram_account_id UUID UNIQUE NOT NULL REFERENCES instagram_accounts(id) ON DELETE CASCADE,

    content_goal TEXT NOT NULL DEFAULT 'reach'
        CHECK (content_goal IN ('reach', 'sales', 'trust', 'engagement', 'awareness')),

    tone_traits TEXT[] NOT NULL DEFAULT '{"friendly"}',
    tone_custom_note TEXT,

    mix_useful INT NOT NULL DEFAULT 40,
    mix_selling INT NOT NULL DEFAULT 25,
    mix_personal INT NOT NULL DEFAULT 20,
    mix_entertaining INT NOT NULL DEFAULT 15,

    format_reels_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    format_reels_pct INT NOT NULL DEFAULT 40,
    format_carousel_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    format_carousel_pct INT NOT NULL DEFAULT 30,
    format_photo_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    format_photo_pct INT NOT NULL DEFAULT 30,

    banned_topics TEXT[] NOT NULL DEFAULT '{}',
    banned_words TEXT[] NOT NULL DEFAULT '{}',
    competitor_names TEXT[] NOT NULL DEFAULT '{}',
    content_restrictions TEXT[] NOT NULL DEFAULT '{}',
    custom_rules TEXT,

    products_services TEXT,
    target_audience TEXT,
    usp TEXT,
    team_members JSONB,
    location_address TEXT,
    working_hours TEXT,
    upcoming_events JSONB,

    content_language TEXT NOT NULL DEFAULT 'ru' CHECK (content_language IN ('ru', 'kz', 'en')),
    posting_frequency TEXT NOT NULL DEFAULT 'daily'
        CHECK (posting_frequency IN ('daily', 'every_other_day', '3_per_week', '2_per_week')),

    niche TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Brand DNA (AI-generated analysis)
CREATE TABLE brand_dna (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instagram_account_id UUID NOT NULL REFERENCES instagram_accounts(id) ON DELETE CASCADE,

    score INT NOT NULL,
    tone TEXT NOT NULL,
    visual_style TEXT,
    strong_topics TEXT[] NOT NULL DEFAULT '{}',
    weak_areas TEXT[] NOT NULL DEFAULT '{}',
    best_formats TEXT[] NOT NULL DEFAULT '{}',
    best_posting_times TEXT[] NOT NULL DEFAULT '{}',
    avg_posting_frequency TEXT,
    hashtag_strategy TEXT,

    strengths JSONB NOT NULL,
    recommendations JSONB NOT NULL,

    raw_analysis JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Content plans
CREATE TABLE content_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instagram_account_id UUID NOT NULL REFERENCES instagram_accounts(id) ON DELETE CASCADE,

    title TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'completed', 'archived')),

    total_slots INT NOT NULL DEFAULT 0,
    approved_slots INT NOT NULL DEFAULT 0,
    published_slots INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Content slots
CREATE TABLE content_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES content_plans(id) ON DELETE CASCADE,

    day_number INT NOT NULL,
    scheduled_date DATE NOT NULL,
    scheduled_time TIME NOT NULL,

    title TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('useful', 'selling', 'personal', 'entertaining')),
    format TEXT NOT NULL CHECK (format IN ('reels', 'carousel', 'photo')),

    brief JSONB NOT NULL,

    caption TEXT NOT NULL,
    hashtags TEXT[] NOT NULL DEFAULT '{}',
    cta TEXT,

    media JSONB NOT NULL DEFAULT '[]',

    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved', 'queued', 'publishing', 'published', 'failed', 'skipped')),
    is_user_content BOOLEAN NOT NULL DEFAULT FALSE,

    published_at TIMESTAMPTZ,
    ig_post_id TEXT,
    ig_post_url TEXT,
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,

    regeneration_count INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(plan_id, day_number)
);

-- Scraped posts
CREATE TABLE scraped_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instagram_account_id UUID NOT NULL REFERENCES instagram_accounts(id) ON DELETE CASCADE,

    ig_post_id TEXT NOT NULL,
    post_type TEXT,
    caption TEXT,
    hashtags TEXT[] DEFAULT '{}',
    likes_count INT,
    comments_count INT,
    posted_at TIMESTAMPTZ,
    thumbnail_url TEXT,

    scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(instagram_account_id, ig_post_id)
);

-- Payments
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL CHECK (provider IN ('dodo', 'kaspi', 'cloudpayments')),
    external_id TEXT NOT NULL,
    amount_cents INT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'cancelled', 'failed')),
    plan TEXT NOT NULL,
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, external_id)
);

-- Indexes
CREATE INDEX idx_ig_accounts_user ON instagram_accounts(user_id);
CREATE INDEX idx_brand_settings_ig ON brand_settings(instagram_account_id);
CREATE INDEX idx_plans_ig ON content_plans(instagram_account_id);
CREATE INDEX idx_slots_plan ON content_slots(plan_id);
CREATE INDEX idx_slots_status_date ON content_slots(status, scheduled_date, scheduled_time);
CREATE INDEX idx_scraped_ig ON scraped_posts(instagram_account_id);
CREATE INDEX idx_payments_user ON payments(user_id);
