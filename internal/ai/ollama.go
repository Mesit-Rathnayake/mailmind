package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaAnalyzer struct {
	client  *http.Client
	model   string
	baseURL string
}

func NewOllamaAnalyzer() *OllamaAnalyzer {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:3b"
	}

	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}

	return &OllamaAnalyzer{
		client:  http.DefaultClient,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (a *OllamaAnalyzer) Analyze(subject, sender, body string) (Analysis, error) {
	// Truncate overly long body to protect local model context window
	cleanBody := strings.TrimSpace(body)
	if len(cleanBody) > 4000 {
		cleanBody = cleanBody[:4000] + "... [truncated]"
	}

	systemPrompt := `You are MailMind, an expert AI email triage assistant.
Analyze the provided email and output ONLY valid JSON matching the exact schema.

Allowed categories:
- LEO: Leo Club, Leo District (e.g. 306), Lions/Leo meetings, installation ceremonies, invitations, notices, leo portal. (ALWAYS use LEO if it mentions Leo/Lions, NEVER PERSONAL or WORK).
- IEEE: IEEE Student Branch, IEEE memberships/renewals, conferences, hackathons, technical webinars.
- UNI: University announcements, Faculty of Engineering notices, lecturers, coursework, exams, academic alerts.
- JOB: Internship opportunities, job offers, LinkedIn job alerts, interview requests, hiring notices.
- SECURITY: Security alerts, password resets, verification codes, 2FA notifications, sign-in alerts.
- FINANCE: Banking, receipts, invoices, payment reminders, billing, subscriptions, refunds.
- WORK: Professional workplace tasks, direct team/client assignments only. (NEVER classify social media or Reddit notifications as WORK).
- PERSONAL: Direct 1-on-1 personal emails from friends/family not related to clubs, work, or automated services.
- PROMOTION: Marketing blasts, product discounts, commercial sales promos, deals.
- SOCIAL: Reddit (r/...), Twitter/X, Instagram, Facebook, YouTube, Discord, Quora, Medium, LinkedIn reactions/connections. (ALWAYS use SOCIAL for Reddit and social digests, NEVER WORK).
- OTHER: Emails that do not clearly fit into any category above.

Allowed priorities:
- CRITICAL: Immediate action required within 24h (security alerts, urgent interview confirmations, payment failures).
- HIGH: Important emails needing attention soon (work assignments, deadlines within a few days).
- MEDIUM: Standard informational or actionable items with no immediate rush.
- LOW: Newsletters, promotions, social digests, Reddit notifications, routine automated notifications.

Rules:
1. category must be exactly one of: LEO, IEEE, UNI, JOB, SECURITY, FINANCE, WORK, PERSONAL, PROMOTION, SOCIAL, OTHER.
2. priority must be exactly one of: CRITICAL, HIGH, MEDIUM, LOW.
3. summary must be a concise, informative 1-2 sentence overview of the email contents. NEVER leave summary empty.
4. action_required is true ONLY if the recipient is required to take an action (e.g. reply, RSVP, pay, confirm attendance).
5. deadline must be an RFC3339 string (e.g. "2026-09-30T17:00:00Z") ONLY if an explicit deadline/date is specified; otherwise null. Do not invent deadlines.

Response JSON format:
{"category":"OTHER","priority":"LOW","summary":"Clear summary of email","action_required":false,"deadline":null}`

	userPrompt := fmt.Sprintf(`Sender: %s
Subject: %s

Email body:
%s`, sender, subject, cleanBody)

	requestBody := struct {
		Model   string                 `json:"model"`
		System  string                 `json:"system"`
		Prompt  string                 `json:"prompt"`
		Format  string                 `json:"format"`
		Stream  bool                   `json:"stream"`
		Options map[string]interface{} `json:"options"`
	}{
		Model:  a.model,
		System: systemPrompt,
		Prompt: userPrompt,
		Format: "json",
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.1,
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return Analysis{}, fmt.Errorf("failed to encode Ollama request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		a.baseURL+"/api/generate",
		bytes.NewReader(payload),
	)
	if err != nil {
		return Analysis{}, fmt.Errorf("failed to create Ollama request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := a.client.Do(request)
	if err != nil {
		return Analysis{}, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Analysis{}, fmt.Errorf("Ollama returned HTTP status %s", response.Status)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Analysis{}, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	output := strings.TrimSpace(result.Response)
	if output == "" {
		return Analysis{}, fmt.Errorf("Ollama returned an empty response")
	}

	var analysis struct {
		Category       string          `json:"category"`
		Priority       string          `json:"priority"`
		Summary        string          `json:"summary"`
		ActionRequired bool            `json:"action_required"`
		Deadline       json.RawMessage `json:"deadline"`
	}

	if err := json.Unmarshal([]byte(output), &analysis); err != nil {
		return Analysis{}, fmt.Errorf("failed to parse Ollama response: %w\nResponse: %s", err, output)
	}

	analysis.Category = strings.ToUpper(strings.TrimSpace(analysis.Category))
	analysis.Priority = strings.ToUpper(strings.TrimSpace(analysis.Priority))
	if analysis.Category == "" {
		analysis.Category = "OTHER"
	}
	if analysis.Priority == "" {
		analysis.Priority = "LOW"
	}

	analysisResult := Analysis{
		Category:       analysis.Category,
		Priority:       analysis.Priority,
		Summary:        analysis.Summary,
		ActionRequired: analysis.ActionRequired,
		Deadline:       parseOllamaDeadline(analysis.Deadline),
	}

	// Refine with deterministic classification rules
	RefineAnalysis(&analysisResult, subject, sender, body)

	if err := validateAnalysis(analysisResult.Category, analysisResult.Priority); err != nil {
		return Analysis{}, err
	}

	return analysisResult, nil
}

func parseOllamaDeadline(raw json.RawMessage) *time.Time {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return parseDeadline(&value)
	}

	var structured struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &structured); err != nil || structured.Value == "" {
		return nil
	}

	return parseDeadline(&structured.Value)
}

func (a *OllamaAnalyzer) DraftReply(subject, sender, body, tone string) (string, error) {
	if tone == "" {
		tone = "professional, concise, and helpful"
	}

	cleanBody := strings.TrimSpace(body)
	if len(cleanBody) > 3000 {
		cleanBody = cleanBody[:3000] + "..."
	}

	systemPrompt := fmt.Sprintf(`You are MailMind, an expert email assistant.
Draft a complete, polite, and contextual email response to the incoming email.
Tone style: %s.
Do not include metadata placeholders or robotic prefixes. Output only the email body response ready to send.`, tone)

	userPrompt := fmt.Sprintf(`Sender: %s
Subject: %s

Original Email:
%s

Draft reply:`, sender, subject, cleanBody)

	requestBody := struct {
		Model   string                 `json:"model"`
		System  string                 `json:"system"`
		Prompt  string                 `json:"prompt"`
		Stream  bool                   `json:"stream"`
		Options map[string]interface{} `json:"options"`
	}{
		Model:  a.model,
		System: systemPrompt,
		Prompt: userPrompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3,
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode draft request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		a.baseURL+"/api/generate",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create draft request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := a.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("draft request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("Ollama returned status %s", response.Status)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimSpace(result.Response), nil
}

