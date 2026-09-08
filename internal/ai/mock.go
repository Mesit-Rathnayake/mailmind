package ai

import "strings"

type MockAnalyzer struct{}

func NewMockAnalyzer() *MockAnalyzer {
	return &MockAnalyzer{}
}

func (m *MockAnalyzer) Analyze(subject, sender, body string) (Analysis, error) {
	text := strings.ToLower(subject + " " + sender + " " + body)

	analysis := Analysis{
		Category:       "OTHER",
		Priority:       "LOW",
		Summary:        subject,
		ActionRequired: false,
	}

	if strings.Contains(text, "meeting") ||
		strings.Contains(text, "interview") ||
		strings.Contains(text, "deadline") {
		analysis.Category = "WORK"
		analysis.Priority = "HIGH"
		analysis.ActionRequired = true
	}

	if strings.Contains(text, "invoice") ||
		strings.Contains(text, "payment") {
		analysis.Category = "FINANCE"
		analysis.Priority = "HIGH"
		analysis.ActionRequired = true
	}

	if strings.Contains(text, "unsubscribe") ||
		strings.Contains(text, "sale") ||
		strings.Contains(text, "discount") {
		analysis.Category = "PROMOTION"
		analysis.Priority = "LOW"
	}

	return analysis, nil
}
