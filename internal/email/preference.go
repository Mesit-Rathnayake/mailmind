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
	Body           string     `json:"body"`
	Category       string     `json:"category"`
	Priority       string     `json:"priority"`
	Summary        string     `json:"summary"`
	ActionRequired bool       `json:"action_required"`
	Deadline       *time.Time `json:"deadline"`
	AttentionScore int        `json:"attention_score"`
	AIProcessedAt  *time.Time `json:"ai_processed_at"`
	IsRead         bool       `json:"is_read"`
	IsReplied      bool       `json:"is_replied"`
	IsArchived     bool       `json:"is_archived"`
	IsStarred      bool       `json:"is_starred"`
	ReadAt         *time.Time `json:"read_at"`
	RepliedAt      *time.Time `json:"replied_at"`
	DraftReply     string     `json:"draft_reply"`
}

type RankedFilter struct {
	TimeFrame string // "12h", "24h", "7d", "30d", "all"
	Status    string // "all", "unread", "read", "replied", "starred", "action", "archived"
	Category  string // "LEO", "IEEE", "UNI", "JOB", "SECURITY", etc. or "ALL"
	Search    string
	Limit     int
	Offset    int
}

type EmailStatusUpdate struct {
	GmailID    string `json:"gmail_id"`
	IsRead     *bool  `json:"is_read,omitempty"`
	IsReplied  *bool  `json:"is_replied,omitempty"`
	IsArchived *bool  `json:"is_archived,omitempty"`
	IsStarred  *bool  `json:"is_starred,omitempty"`
	DraftReply *string `json:"draft_reply,omitempty"`
}

type EmailStats struct {
	Total          int `json:"total"`
	Last12Hours    int `json:"last_12h"`
	Last24Hours    int `json:"last_24h"`
	Last7Days      int `json:"last_7d"`
	Last30Days     int `json:"last_30d"`
	Unread         int `json:"unread"`
	Read           int `json:"read"`
	Replied        int `json:"replied"`
	Starred        int `json:"starred"`
	ActionRequired int `json:"action_required"`
	Categories     map[string]int `json:"categories"`
}
