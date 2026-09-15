package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type Analysis struct {
	Category       string
	Priority       string
	Summary        string
	ActionRequired bool
	Deadline       *time.Time
	AttentionScore int
}

type Analyzer interface {
	Analyze(subject, sender, body string) (Analysis, error)
	DraftReply(subject, sender, body, tone string) (string, error)
}

func NewAnalyzerFromEnv(ctx context.Context) (Analyzer, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))

	switch provider {
	case "", "ollama":
		return NewOllamaAnalyzer(), nil
	case "gemini":
		return NewGeminiAnalyzer(ctx)
	case "openai":
		return NewOpenAIAnalyzer()
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER %q", os.Getenv("AI_PROVIDER"))
	}
}
