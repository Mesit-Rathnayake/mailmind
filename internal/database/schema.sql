CREATE TABLE IF NOT EXISTS emails (
    id BIGSERIAL PRIMARY KEY,
    gmail_id TEXT NOT NULL UNIQUE,
    thread_id TEXT NOT NULL,
    sender TEXT NOT NULL,
    recipient TEXT,
    subject TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    snippet TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);