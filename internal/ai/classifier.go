package ai

import (
	"strings"
)

// MatchCategoryRules returns a high-confidence category and priority based on deterministic email metadata.
func MatchCategoryRules(subject, sender, body string) (string, string, bool) {
	lowerSender := strings.ToLower(sender)
	lowerSubject := strings.ToLower(subject)
	lowerBody := strings.ToLower(body)

	fullHeader := lowerSender + " " + lowerSubject

	// 1. LEO CLUB / DISTRICT / LIONS
	if strings.Contains(fullHeader, "leo") ||
		strings.Contains(lowerSender, "leoclub") ||
		strings.Contains(lowerSender, "leodistrict") ||
		strings.Contains(lowerSubject, "leo club") ||
		strings.Contains(lowerSubject, "leo district") ||
		strings.Contains(lowerSubject, "leoistic") ||
		strings.Contains(lowerSubject, "leo portal") ||
		strings.Contains(lowerSubject, "multiple district 306") ||
		strings.Contains(lowerSubject, "district 306") ||
		strings.Contains(lowerSubject, "lions club") ||
		strings.Contains(lowerBody, "leo club") ||
		strings.Contains(lowerBody, "leo district") ||
		strings.Contains(lowerBody, "leoistic year") ||
		strings.Contains(lowerBody, "fellow leos") ||
		strings.Contains(lowerBody, "dear leos") ||
		strings.Contains(lowerBody, "leoism") {
		return "LEO", "", true
	}

	// 2. SOCIAL (Reddit, Twitter/X, Instagram, Facebook, YouTube, Discord, Quora, Medium, etc.)
	if strings.Contains(lowerSender, "redditmail.com") ||
		strings.Contains(lowerSender, "reddit.com") ||
		strings.Contains(lowerSender, "notifications@reddit.com") ||
		strings.Contains(lowerSubject, "r/") ||
		strings.Contains(lowerSubject, "reddit") ||
		strings.Contains(lowerSender, "twitter.com") ||
		strings.Contains(lowerSender, "x.com") ||
		strings.Contains(lowerSender, "instagram.com") ||
		strings.Contains(lowerSender, "facebookmail.com") ||
		strings.Contains(lowerSender, "youtube.com") ||
		strings.Contains(lowerSender, "discord.com") ||
		strings.Contains(lowerSender, "quora.com") ||
		strings.Contains(lowerSender, "medium.com") ||
		strings.Contains(lowerSender, "pinterest.com") ||
		strings.Contains(lowerSender, "notifications@github.com") ||
		strings.Contains(lowerSender, "invitations@linkedin.com") ||
		strings.Contains(lowerSender, "updates@linkedin.com") ||
		strings.Contains(lowerSender, "messaging-digest-noreply@linkedin.com") {
		return "SOCIAL", "LOW", true
	}

	// 3. IEEE
	if strings.Contains(lowerSender, "ieee.org") ||
		strings.Contains(lowerSender, "ieee") ||
		strings.Contains(lowerSubject, "ieee") ||
		strings.Contains(lowerSubject, "ieeextreme") ||
		strings.Contains(lowerSubject, "student branch") ||
		strings.Contains(lowerBody, "ieee student branch") ||
		strings.Contains(lowerBody, "ieee membership") {
		return "IEEE", "", true
	}

	// 4. SECURITY & AUTH (Google Security, Microsoft, OTP, 2FA, Password resets)
	if strings.Contains(lowerSender, "accounts.google.com") ||
		strings.Contains(lowerSender, "accountprotection.microsoft.com") ||
		strings.Contains(lowerSubject, "security alert") ||
		strings.Contains(lowerSubject, "verification code") ||
		strings.Contains(lowerSubject, "password reset") ||
		strings.Contains(lowerSubject, "2-step verification") ||
		strings.Contains(lowerSubject, "new sign-in on") ||
		strings.Contains(lowerSubject, "suspicious activity") ||
		strings.Contains(lowerSubject, "one-time passcode") ||
		strings.Contains(lowerSubject, "your otp") {
		return "SECURITY", "CRITICAL", true
	}

	// 5. JOB & INTERNSHIPS (Himalayas, LinkedIn Job Alerts, Indeed, Glassdoor, etc.)
	if strings.Contains(lowerSender, "himalayas.app") ||
		strings.Contains(lowerSender, "himalayas") ||
		strings.Contains(lowerSender, "jobalerts-noreply@linkedin.com") ||
		strings.Contains(lowerSender, "indeed.com") ||
		strings.Contains(lowerSender, "internshala.com") ||
		strings.Contains(lowerSender, "glassdoor.com") ||
		strings.Contains(lowerSender, "wellfound.com") ||
		strings.Contains(lowerSubject, "job alert") ||
		strings.Contains(lowerSubject, "interview invitation") ||
		strings.Contains(lowerSubject, "job application") ||
		strings.Contains(lowerSubject, "application status") ||
		strings.Contains(lowerSubject, "we are hiring") ||
		strings.Contains(lowerSubject, "job opportunities") {
		return "JOB", "", true
	}

	// 6. FINANCE (Stripe, PayPal, Banks, Billing, Receipts)
	if strings.Contains(lowerSender, "paypal.com") ||
		strings.Contains(lowerSender, "stripe.com") ||
		strings.Contains(lowerSender, "billing@") ||
		strings.Contains(lowerSender, "invoicing@") ||
		strings.Contains(lowerSender, "payoneer.com") ||
		strings.Contains(lowerSender, "wise.com") ||
		strings.Contains(lowerSubject, "receipt for your payment") ||
		strings.Contains(lowerSubject, "invoice #") ||
		strings.Contains(lowerSubject, "payment confirmation") ||
		strings.Contains(lowerSubject, "billing statement") ||
		strings.Contains(lowerSubject, "transaction alert") {
		return "FINANCE", "", true
	}

	// 7. UNIVERSITY / ACADEMIC
	if strings.Contains(lowerSender, ".edu") ||
		strings.Contains(lowerSender, ".ac.lk") ||
		strings.Contains(lowerSubject, "faculty of engineering") ||
		strings.Contains(lowerSubject, "semester exam") ||
		strings.Contains(lowerSubject, "course registration") ||
		strings.Contains(lowerSubject, "lecture schedule") ||
		strings.Contains(lowerSubject, "coursework submission") {
		return "UNI", "", true
	}

	return "", "", false
}

// RefineAnalysis ensures deterministic rules override or refine AI classification errors.
func RefineAnalysis(analysis *Analysis, subject, sender, body string) {
	if cat, prio, ok := MatchCategoryRules(subject, sender, body); ok {
		if cat != "" {
			analysis.Category = cat
		}
		if prio != "" && (analysis.Priority == "" || analysis.Priority == "LOW") {
			analysis.Priority = prio
		}
		// If social digest (like Reddit), ensure priority isn't mistakenly CRITICAL/HIGH
		if cat == "SOCIAL" && (analysis.Priority == "CRITICAL" || analysis.Priority == "HIGH") {
			analysis.Priority = "LOW"
		}
	}

	// Default fallback guards
	if analysis.Category == "" {
		analysis.Category = "OTHER"
	}
	if analysis.Priority == "" {
		analysis.Priority = "LOW"
	}
}
