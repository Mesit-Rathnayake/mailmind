package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connectionString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{pool: pool}
	if err := db.Migrate(ctx); err != nil {
		log.Printf("Warning: Database auto-migration error: %v", err)
	}

	return db, nil
}

func (db *DB) Migrate(ctx context.Context) error {
	migrations := `
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
			fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			category TEXT,
			priority TEXT,
			summary TEXT,
			action_required BOOLEAN DEFAULT FALSE,
			deadline TIMESTAMPTZ,
			ai_processed_at TIMESTAMPTZ,
			ai_model TEXT,
			analysis_version INT DEFAULT 1,
			processed BOOLEAN NOT NULL DEFAULT FALSE,
			attention_score DOUBLE PRECISION,
			scored_at TIMESTAMPTZ,
			is_read BOOLEAN NOT NULL DEFAULT FALSE,
			is_replied BOOLEAN NOT NULL DEFAULT FALSE,
			is_archived BOOLEAN NOT NULL DEFAULT FALSE,
			is_starred BOOLEAN NOT NULL DEFAULT FALSE,
			read_at TIMESTAMPTZ,
			replied_at TIMESTAMPTZ,
			draft_reply TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_replied BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_starred BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS replied_at TIMESTAMPTZ;
		ALTER TABLE emails ADD COLUMN IF NOT EXISTS draft_reply TEXT;

		CREATE INDEX IF NOT EXISTS idx_emails_ai_unprocessed ON emails (received_at DESC) WHERE ai_processed_at IS NULL;
		CREATE INDEX IF NOT EXISTS idx_emails_attention_score ON emails (attention_score DESC NULLS LAST, received_at DESC);
		CREATE INDEX IF NOT EXISTS idx_emails_received_at ON emails (received_at DESC);

		CREATE TABLE IF NOT EXISTS user_preferences (
			id BIGSERIAL PRIMARY KEY,
			target_type TEXT NOT NULL,
			target_value TEXT NOT NULL,
			score_modifier INT NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(target_type, target_value)
		);

		INSERT INTO user_preferences (target_type, target_value, score_modifier, description)
		VALUES 
			('CATEGORY', 'LEO', 45, 'Leo Club priority emails'),
			('CATEGORY', 'SECURITY', 35, 'Security alerts and OTPs'),
			('CATEGORY', 'UNI', 30, 'University and academic updates'),
			('CATEGORY', 'IEEE', 25, 'IEEE branch notifications'),
			('CATEGORY', 'JOB', 25, 'Internship and job opportunities'),
			('CATEGORY', 'WORK', 15, 'Work related emails'),
			('CATEGORY', 'PROMOTION', -25, 'Marketing and newsletter promotions')
		ON CONFLICT (target_type, target_value) DO NOTHING;
	`

	_, err := db.pool.Exec(ctx, migrations)
	if err != nil {
		return fmt.Errorf("failed executing migrations: %w", err)
	}

	return nil
}

func (db *DB) Close(ctx context.Context) error {
	db.pool.Close()
	return nil
}
