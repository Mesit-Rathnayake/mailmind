package email

import "time"

type Email struct {
	ID       string
	ThreadID string
	From     string
	To       string
	Subject  string
	Date     time.Time
	Snippet  string
	Body     string
}
