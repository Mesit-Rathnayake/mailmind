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

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_emails_ai_unprocessed 
    ON emails (received_at DESC) 
    WHERE ai_processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_emails_unscored 
    ON emails (ai_processed_at DESC) 
    WHERE ai_processed_at IS NOT NULL AND scored_at IS NULL;