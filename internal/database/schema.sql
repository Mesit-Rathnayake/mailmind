-- =========================================================================
-- MailMind Cloud PostgreSQL Schema
-- Compatible with Neon, Supabase, and local PostgreSQL
-- =========================================================================

-- 1. Emails Table
CREATE TABLE IF NOT EXISTS emails (
    id BIGSERIAL PRIMARY KEY,
    gmail_id TEXT NOT NULL UNIQUE,
    thread_id TEXT NOT NULL,
    sender TEXT NOT NULL,
    recipient TEXT,
    subject TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    snippet TEXT,
    body TEXT,
    
    -- Ingestion
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- AI Triage
    category TEXT,
    priority TEXT,
    summary TEXT,
    action_required BOOLEAN DEFAULT FALSE,
    deadline TIMESTAMPTZ,
    ai_processed_at TIMESTAMPTZ,
    ai_model TEXT,
    analysis_version INT DEFAULT 1,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Scoring & Ranking
    attention_score DOUBLE PRECISION,
    scored_at TIMESTAMPTZ,

    -- Status & Interaction Tracking
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    is_replied BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    replied_at TIMESTAMPTZ,
    draft_reply TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migration for existing tables
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_replied BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_starred BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS replied_at TIMESTAMPTZ;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS draft_reply TEXT;

-- Indexes for fast querying & worker queue processing
CREATE INDEX IF NOT EXISTS idx_emails_ai_unprocessed 
    ON emails (received_at DESC) 
    WHERE ai_processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_emails_attention_score 
    ON emails (attention_score DESC NULLS LAST, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_emails_received_at
    ON emails (received_at DESC);

CREATE INDEX IF NOT EXISTS idx_emails_status
    ON emails (is_read, is_replied, is_archived, is_starred);

-- 2. User Preferences Table
CREATE TABLE IF NOT EXISTS user_preferences (
    id BIGSERIAL PRIMARY KEY,
    rule_type TEXT NOT NULL,       -- 'CATEGORY', 'SENDER', 'DOMAIN'
    target_value TEXT NOT NULL,    -- e.g., 'LEO', 'UNI', 'IEEE', 'linkedin.com'
    score_modifier INT NOT NULL,   -- e.g., +45, +30, -25
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(rule_type, target_value)
);

-- Seed Default High-Priority Rules (Leo Club, IEEE, Uni, Security)
INSERT INTO user_preferences (rule_type, target_value, score_modifier, description)
VALUES 
    ('CATEGORY', 'LEO', 45, 'Leo Club priority emails'),
    ('CATEGORY', 'SECURITY', 35, 'Security alerts and OTPs'),
    ('CATEGORY', 'UNI', 30, 'University and academic updates'),
    ('CATEGORY', 'IEEE', 25, 'IEEE branch notifications'),
    ('CATEGORY', 'JOB', 25, 'Internship and job opportunities'),
    ('CATEGORY', 'WORK', 15, 'Work related emails'),
    ('CATEGORY', 'PROMOTION', -25, 'Marketing and newsletter promotions')
ON CONFLICT (rule_type, target_value) DO NOTHING;