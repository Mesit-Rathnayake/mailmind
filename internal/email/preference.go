package email

import "time"

type Preference struct {
	ID            int64     `json:"id"`
	TargetType    string    `json:"target_type"` // 'CATEGORY', 'KEYWORD', 'SENDER', 'DOMAIN'
	TargetValue   string    `json:"target_value"`
	ScoreModifier int       `json:"score_modifier"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type RankedEmailView struct {
	GmailID        string     `json:"gmail_id"`
	ThreadID       string     `json:"thread_id"`
	Sender         string     `json:"sender"`
	Recipient      string     `json:"recipient"`
	Subject        string     `json:"subject"`
	ReceivedAt     time.Time  `json:"received_at"`
	Snippet        string     `json:"snippet"`
	Category       string     `json:"category"`
	Priority       string     `json:"priority"`
	Summary        string     `json:"summary"`
	ActionRequired bool       `json:"action_required"`
	Deadline       *time.Time `json:"deadline"`
	AttentionScore int        `json:"attention_score"`
	AIProcessedAt  *time.Time `json:"ai_processed_at"`
}
