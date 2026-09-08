package database

import (
	"context"
	"fmt"

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
		WHERE processed = FALSE
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
