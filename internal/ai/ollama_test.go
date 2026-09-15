package ai

import (
	"testing"
)

func TestOllamaAnalyzer(t *testing.T) {
	analyzer := NewOllamaAnalyzer()

	testCases := []struct {
		name             string
		subject          string
		sender           string
		body             string
		expectedCategory string
	}{
		{
			name:             "Security alert",
			subject:          "Urgent: Security Alert - Password reset requested",
			sender:           "security@example.com",
			body:             "We detected a password reset request for your account. If this was not you, please secure your account immediately by clicking the link before 2026-09-16T12:00:00Z.",
			expectedCategory: "SECURITY",
		},
		{
			name:             "IEEE Renewal",
			subject:          "IEEE Membership(s) and Subscription(s) Automatic Renewal reminder",
			sender:           "renewals@ieee.org",
			body:             "Dear Member, Your IEEE membership is scheduled for automatic renewal on October 1st. Please review your subscription details.",
			expectedCategory: "IEEE",
		},
		{
			name:             "University Exam Notice",
			subject:          "Faculty of Engineering - End Semester Examination Schedule",
			sender:           "dean-eng@university.ac.lk",
			body:             "All undergraduate students are hereby informed that end semester examinations will commence on 2026-10-15T09:00:00Z.",
			expectedCategory: "UNI",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tc.subject, tc.sender, tc.body)
			if err != nil {
				t.Fatalf("Ollama analysis failed: %v", err)
			}

			t.Logf("[%s] Category: %s | Priority: %s | Summary: %s | ActionRequired: %v | Deadline: %v",
				tc.name, result.Category, result.Priority, result.Summary, result.ActionRequired, result.Deadline)

			if result.Category != tc.expectedCategory {
				t.Errorf("expected category %s, got %s", tc.expectedCategory, result.Category)
			}
			if result.Summary == "" {
				t.Errorf("summary should not be empty")
			}
		})
	}
}
