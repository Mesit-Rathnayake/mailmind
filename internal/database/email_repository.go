package database

import (
	"context"
	"fmt"
	"strings"
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
			body,
			is_read,
			is_starred
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (gmail_id) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			sender = EXCLUDED.sender,
			recipient = EXCLUDED.recipient,
			subject = EXCLUDED.subject,
			received_at = EXCLUDED.received_at,
			snippet = EXCLUDED.snippet,
			body = EXCLUDED.body
	`

	_, err := db.pool.Exec(
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
		e.IsRead,
		e.IsStarred,
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
			COALESCE(body, ''),
			is_read,
			is_starred
		FROM emails
		WHERE ai_processed_at IS NULL
		ORDER BY received_at DESC
		LIMIT $1
	`

	rows, err := db.pool.Query(ctx, query, limit)
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
			&e.IsRead,
			&e.IsStarred,
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

	_, err := db.pool.Exec(ctx, query, gmailID)
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

	_, err := db.pool.Exec(
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

func (db *DB) UpdateEmailStatus(ctx context.Context, update email.EmailStatusUpdate) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if update.IsRead != nil {
		sets = append(sets, fmt.Sprintf("is_read = $%d", argIdx))
		args = append(args, *update.IsRead)
		argIdx++

		if *update.IsRead {
			sets = append(sets, "read_at = NOW()")
		} else {
			sets = append(sets, "read_at = NULL")
		}
	}

	if update.IsReplied != nil {
		sets = append(sets, fmt.Sprintf("is_replied = $%d", argIdx))
		args = append(args, *update.IsReplied)
		argIdx++

		if *update.IsReplied {
			sets = append(sets, "replied_at = NOW()")
		} else {
			sets = append(sets, "replied_at = NULL")
		}
	}

	if update.IsArchived != nil {
		sets = append(sets, fmt.Sprintf("is_archived = $%d", argIdx))
		args = append(args, *update.IsArchived)
		argIdx++
	}

	if update.IsStarred != nil {
		sets = append(sets, fmt.Sprintf("is_starred = $%d", argIdx))
		args = append(args, *update.IsStarred)
		argIdx++
	}

	if update.DraftReply != nil {
		sets = append(sets, fmt.Sprintf("draft_reply = $%d", argIdx))
		args = append(args, *update.DraftReply)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE emails SET %s WHERE gmail_id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, update.GmailID)

	_, err := db.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update status for email %s: %w", update.GmailID, err)
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

	rows, err := db.pool.Query(ctx, query)
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

func (db *DB) GetRankedEmails(ctx context.Context, filter email.RankedFilter) ([]email.RankedEmailView, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, "ai_processed_at IS NOT NULL")

	// 1. Time Frame Filter
	switch strings.ToLower(filter.TimeFrame) {
	case "12h":
		whereClauses = append(whereClauses, "received_at >= NOW() - INTERVAL '12 hours'")
	case "24h", "1d":
		whereClauses = append(whereClauses, "received_at >= NOW() - INTERVAL '24 hours'")
	case "7d", "1w", "week":
		whereClauses = append(whereClauses, "received_at >= NOW() - INTERVAL '7 days'")
	case "30d", "1m", "month":
		whereClauses = append(whereClauses, "received_at >= NOW() - INTERVAL '30 days'")
	}

	// 2. Status Filter
	switch strings.ToLower(filter.Status) {
	case "unread":
		whereClauses = append(whereClauses, "is_read = FALSE AND is_archived = FALSE")
	case "read":
		whereClauses = append(whereClauses, "is_read = TRUE AND is_archived = FALSE")
	case "replied":
		whereClauses = append(whereClauses, "is_replied = TRUE")
	case "starred":
		whereClauses = append(whereClauses, "is_starred = TRUE")
	case "action":
		whereClauses = append(whereClauses, "action_required = TRUE AND is_archived = FALSE")
	case "archived":
		whereClauses = append(whereClauses, "is_archived = TRUE")
	default:
		// Normal inbox view excludes archived
		whereClauses = append(whereClauses, "is_archived = FALSE")
	}

	// 3. Category Filter
	if filter.Category != "" && !strings.EqualFold(filter.Category, "ALL") {
		whereClauses = append(whereClauses, fmt.Sprintf("UPPER(category) = UPPER($%d)", argIdx))
		args = append(args, filter.Category)
		argIdx++
	}

	// 4. Search Filter
	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(subject ILIKE $%d OR sender ILIKE $%d OR summary ILIKE $%d OR snippet ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT
			gmail_id,
			thread_id,
			sender,
			COALESCE(recipient, ''),
			COALESCE(subject, ''),
			received_at,
			COALESCE(snippet, ''),
			COALESCE(body, ''),
			COALESCE(category, ''),
			COALESCE(priority, ''),
			COALESCE(summary, ''),
			COALESCE(action_required, FALSE),
			deadline,
			COALESCE(attention_score, 0),
			ai_processed_at,
			is_read,
			is_replied,
			is_archived,
			is_starred,
			read_at,
			replied_at,
			COALESCE(draft_reply, '')
		FROM emails
		WHERE %s
		ORDER BY attention_score DESC, received_at DESC
		LIMIT $%d
	`, strings.Join(whereClauses, " AND "), argIdx)

	args = append(args, limit)

	rows, err := db.pool.Query(ctx, query, args...)
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
			&r.Body,
			&r.Category,
			&r.Priority,
			&r.Summary,
			&r.ActionRequired,
			&r.Deadline,
			&r.AttentionScore,
			&r.AIProcessedAt,
			&r.IsRead,
			&r.IsReplied,
			&r.IsArchived,
			&r.IsStarred,
			&r.ReadAt,
			&r.RepliedAt,
			&r.DraftReply,
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

func (db *DB) GetEmailStats(ctx context.Context) (email.EmailStats, error) {
	stats := email.EmailStats{
		Categories: make(map[string]int),
	}

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE received_at >= NOW() - INTERVAL '12 hours') as last_12h,
			COUNT(*) FILTER (WHERE received_at >= NOW() - INTERVAL '24 hours') as last_24h,
			COUNT(*) FILTER (WHERE received_at >= NOW() - INTERVAL '7 days') as last_7d,
			COUNT(*) FILTER (WHERE received_at >= NOW() - INTERVAL '30 days') as last_30d,
			COUNT(*) FILTER (WHERE is_read = FALSE AND is_archived = FALSE) as unread,
			COUNT(*) FILTER (WHERE is_read = TRUE AND is_archived = FALSE) as read,
			COUNT(*) FILTER (WHERE is_replied = TRUE) as replied,
			COUNT(*) FILTER (WHERE is_starred = TRUE) as starred,
			COUNT(*) FILTER (WHERE action_required = TRUE AND is_archived = FALSE) as action_required
		FROM emails
		WHERE ai_processed_at IS NOT NULL
	`

	err := db.pool.QueryRow(ctx, query).Scan(
		&stats.Total,
		&stats.Last12Hours,
		&stats.Last24Hours,
		&stats.Last7Days,
		&stats.Last30Days,
		&stats.Unread,
		&stats.Read,
		&stats.Replied,
		&stats.Starred,
		&stats.ActionRequired,
	)
	if err != nil {
		return stats, fmt.Errorf("failed to query email stats: %w", err)
	}

	catQuery := `
		SELECT UPPER(category), COUNT(*)
		FROM emails
		WHERE ai_processed_at IS NOT NULL AND is_archived = FALSE AND category IS NOT NULL AND category != ''
		GROUP BY UPPER(category)
	`
	rows, err := db.pool.Query(ctx, catQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if err := rows.Scan(&cat, &count); err == nil {
				stats.Categories[cat] = count
			}
		}
	}

	return stats, nil
}
