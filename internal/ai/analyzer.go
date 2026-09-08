package ai

import "time"

type Analysis struct {
	Category       string
	Priority       string
	Summary        string
	ActionRequired bool
	Deadline       *time.Time
}

type Analyzer interface {
	Analyze(subject, sender, body string) (Analysis, error)
}
