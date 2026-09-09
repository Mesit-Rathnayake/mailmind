package ai

import (
	"strings"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/email"
)

func CalculateAttentionScore(e email.Email, analysis Analysis, prefs []email.Preference) int {
	score := 0

	// --------------------------------------------------
	// Base score from AI priority
	// --------------------------------------------------

	switch analysis.Priority {
	case "CRITICAL":
		score += 100
	case "HIGH":
		score += 70
	case "MEDIUM":
		score += 40
	case "LOW":
		score += 10
	}

	// --------------------------------------------------
	// Action required
	// --------------------------------------------------

	if analysis.ActionRequired {
		score += 20
	}

	// --------------------------------------------------
	// Deadline
	// --------------------------------------------------

	if analysis.Deadline != nil {
		hoursUntilDeadline := time.Until(*analysis.Deadline).Hours()

		switch {
		case hoursUntilDeadline <= 24:
			score += 30
		case hoursUntilDeadline <= 72:
			score += 20
		case hoursUntilDeadline <= 168:
			score += 10
		}
	}

	// --------------------------------------------------
	// Recency signal
	// --------------------------------------------------

	if !e.Date.IsZero() {
		ageInHours := time.Since(e.Date).Hours()

		switch {
		case ageInHours < 1:
			score += 15
		case ageInHours < 6:
			score += 10
		case ageInHours < 24:
			score += 5
		}
	}

	// --------------------------------------------------
	// Dynamic User Preferences Matching
	// --------------------------------------------------

	lowerSubject := strings.ToLower(e.Subject)
	lowerSender := strings.ToLower(e.From)
	lowerBody := strings.ToLower(e.Body)
	fullText := lowerSubject + " " + lowerSender + " " + lowerBody

	for _, p := range prefs {
		val := strings.ToLower(p.TargetValue)

		switch p.TargetType {
		case "CATEGORY":
			if strings.EqualFold(analysis.Category, p.TargetValue) {
				score += p.ScoreModifier
			}
		case "SENDER":
			if strings.Contains(lowerSender, val) {
				score += p.ScoreModifier
			}
		case "KEYWORD":
			if strings.Contains(fullText, val) {
				score += p.ScoreModifier
			}
		}
	}

	// --------------------------------------------------
	// Prevent negative scores
	// --------------------------------------------------

	if score < 0 {
		score = 0
	}

	return score
}
