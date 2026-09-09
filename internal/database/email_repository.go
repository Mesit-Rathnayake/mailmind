package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/email"
)

func (db *DB) SaveEmail(ctx context.Context, e email.Email) error {
	query := `
		INSERT INTO emails (
			gmail_id,
			thread_id,
			sender,
			recipient,
			subject,
			received_at,
			snippet,
			body
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (gmail_id) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			sender = EXCLUDED.sender,
			recipient = EXCLUDED.recipient,
			subject = EXCLUDED.subject,
			received_at = EXCLUDED.received_at,
			snippet = EXCLUDED.snippet,
			body = EXCLUDED.body
	`

	_, err := db.conn.Exec(
		ctx,
		query,
		e.ID,
		e.ThreadID,
		e.From,
		e.To,
		e.Subject,
		e.Date,
		e.Snippet,
		e.Body,
	)

	if err != nil {
		return fmt.Errorf("failed to save email %s: %w", e.ID, err)
	}

	return nil
}

func (db *DB) GetUnprocessedEmails(ctx context.Context, limit int) ([]email.Email, error) {
	query := `
		SELECT
			gmail_id,
			thread_id,
			sender,
			COALESCE(recipient, ''),
			COALESCE(subject, ''),
			received_at,
			COALESCE(snippet, ''),
			COALESCE(body, '')
		FROM emails
		WHERE ai_processed_at IS NULL
		ORDER BY received_at DESC
		LIMIT $1
	`

	rows, err := db.conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed emails: %w", err)
	}
	defer rows.Close()

	var emails []email.Email

	for rows.Next() {
		var e email.Email

		if err := rows.Scan(
			&e.ID,
			&e.ThreadID,
			&e.From,
			&e.To,
			&e.Subject,
			&e.Date,
			&e.Snippet,
			&e.Body,
		); err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		emails = append(emails, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading emails: %w", err)
	}

	return emails, nil
}

func (db *DB) MarkEmailProcessed(ctx context.Context, gmailID string) error {
	query := `
		UPDATE emails
		SET processed = TRUE
		WHERE gmail_id = $1
	`

	_, err := db.conn.Exec(ctx, query, gmailID)
	if err != nil {
		return fmt.Errorf("failed to mark email %s as processed: %w", gmailID, err)
	}

	return nil
}

func (db *DB) SaveAnalysis(
	ctx context.Context,
	gmailID string,
	category string,
	priority string,
	summary string,
	actionRequired bool,
	deadline *time.Time,
	attentionScore int,
) error {
	query := `
		UPDATE emails
		SET
			category = $1,
			priority = $2,
			summary = $3,
			action_required = $4,
			deadline = $5,
			attention_score = $6,
			ai_processed_at = NOW(),
			processed = TRUE
		WHERE gmail_id = $7
	`

	_, err := db.conn.Exec(
		ctx,
		query,
		category,
		priority,
		summary,
		actionRequired,
		deadline,
		attentionScore,
		gmailID,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to save AI analysis for email %s: %w",
			gmailID,
			err,
		)
	}

	return nil
}

func (db *DB) SaveAttentionScore(
	ctx context.Context,
	gmailID string,
	score float64,
) error {
	query := `
		UPDATE emails
		SET
			attention_score = $1,
			scored_at = NOW()
		WHERE gmail_id = $2
	`

	_, err := db.conn.Exec(ctx, query, score, gmailID)
	if err != nil {
		return fmt.Errorf("failed to save attention score for email %s: %w", gmailID, err)
	}

	return nil
}

func (db *DB) GetUserPreferences(ctx context.Context) ([]email.Preference, error) {
	query := `
		SELECT
			id,
			target_type,
			target_value,
			score_modifier,
			COALESCE(description, ''),
			created_at
		FROM user_preferences
		ORDER BY id ASC
	`

	rows, err := db.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user preferences: %w", err)
	}
	defer rows.Close()

	var prefs []email.Preference
	for rows.Next() {
		var p email.Preference
		if err := rows.Scan(
			&p.ID,
			&p.TargetType,
			&p.TargetValue,
			&p.ScoreModifier,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}
		prefs = append(prefs, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading user preferences: %w", err)
	}

	return prefs, nil
}

func (db *DB) GetRankedEmails(ctx context.Context, limit int) ([]email.RankedEmailView, error) {
	query := `
		SELECT
			gmail_id,
			thread_id,
			sender,
			COALESCE(recipient, ''),
			COALESCE(subject, ''),
			received_at,
			COALESCE(snippet, ''),
			COALESCE(category, ''),
			COALESCE(priority, ''),
			COALESCE(summary, ''),
			COALESCE(action_required, FALSE),
			deadline,
			COALESCE(attention_score, 0),
			ai_processed_at
		FROM emails
		WHERE ai_processed_at IS NOT NULL
		ORDER BY attention_score DESC, received_at DESC
		LIMIT $1
	`

	rows, err := db.conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ranked emails: %w", err)
	}
	defer rows.Close()

	var ranked []email.RankedEmailView
	for rows.Next() {
		var r email.RankedEmailView
		if err := rows.Scan(
			&r.GmailID,
			&r.ThreadID,
			&r.Sender,
			&r.Recipient,
			&r.Subject,
			&r.ReceivedAt,
			&r.Snippet,
			&r.Category,
			&r.Priority,
			&r.Summary,
			&r.ActionRequired,
			&r.Deadline,
			&r.AttentionScore,
			&r.AIProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ranked email: %w", err)
		}
		ranked = append(ranked, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading ranked emails: %w", err)
	}

	return ranked, nil
}
