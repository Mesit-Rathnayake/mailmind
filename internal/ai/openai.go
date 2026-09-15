package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAIAnalyzer struct {
	client *openai.Client
	model  string
}

func NewOpenAIAnalyzer() (*OpenAIAnalyzer, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5.6"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &OpenAIAnalyzer{
		client: &client,
		model:  model,
	}, nil
}

func (a *OpenAIAnalyzer) Analyze(
	subject string,
	sender string,
	body string,
) (Analysis, error) {

	instructions := `
You are MailMind, an intelligent email triage assistant.

Analyze the email and return ONLY valid JSON.

Allowed categories:
LEO, IEEE, UNI, JOB, SECURITY, FINANCE, WORK, PERSONAL, PROMOTION, SOCIAL, OTHER

Category guidance:
- LEO: Leo Club, Leo District (e.g. 306), Lions/Leo meetings, installation ceremonies, invitations, notices, leo portal. (ALWAYS use LEO if it mentions Leo/Lions, NEVER PERSONAL or WORK).
- IEEE: IEEE Student Branch, IEEE memberships/renewals, conferences, hackathons, technical webinars.
- UNI: University announcements, Faculty of Engineering notices, lecturers, coursework, exams, academic alerts.
- JOB: Internship opportunities, job offers, LinkedIn job alerts, interview requests, hiring notices.
- SECURITY: Security alerts, password resets, verification codes, 2FA notifications, sign-in alerts.
- FINANCE: Banking, receipts, invoices, payment reminders, billing, subscriptions, refunds.
- WORK: Professional workplace tasks, direct team/client assignments only. (NEVER classify social media or Reddit notifications as WORK).
- PERSONAL: Direct personal emails from friends or family not related to clubs or automated services.
- PROMOTION: Marketing blasts, product discounts, commercial sales promos, deals.
- SOCIAL: Reddit (r/...), Twitter/X, Instagram, Facebook, YouTube, Discord, Quora, Medium, LinkedIn reactions/connections. (ALWAYS use SOCIAL for Reddit, NEVER WORK).
- OTHER: Emails that do not clearly fit into any category above.

Allowed priorities:
CRITICAL, HIGH, MEDIUM, LOW

Rules:
- category must be exactly one allowed category.
- priority must be exactly one allowed priority.
- summary must be concise and factual.
- action_required is true only when the recipient needs to do something.
- deadline must be an RFC3339 timestamp when a deadline is explicitly mentioned or clearly implied.
- deadline must be null when there is no meaningful deadline.
- Do not invent deadlines.
- Do not include markdown.
- Do not include explanations outside the JSON.

Return exactly:

{
  "category": "WORK",
  "priority": "HIGH",
  "summary": "Short summary",
  "action_required": true,
  "deadline": null
}
`

	input := fmt.Sprintf(
		"%s\n\nSender: %s\nSubject: %s\n\nEmail body:\n%s",
		instructions,
		sender,
		subject,
		body,
	)

	ctx := context.Background()

	response, err := a.client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Model: shared.ResponsesModel(a.model),
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(input),
			},
			MaxOutputTokens: openai.Int(500),
		},
	)

	if err != nil {
		return Analysis{}, fmt.Errorf("OpenAI request failed: %w", err)
	}

	output := strings.TrimSpace(response.OutputText())

	if output == "" {
		return Analysis{}, fmt.Errorf("OpenAI returned an empty response")
	}

	var result struct {
		Category       string  `json:"category"`
		Priority       string  `json:"priority"`
		Summary        string  `json:"summary"`
		ActionRequired bool    `json:"action_required"`
		Deadline       *string `json:"deadline"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return Analysis{}, fmt.Errorf(
			"failed to parse OpenAI response: %w\nResponse: %s",
			err,
			output,
		)
	}

	analysisResult := Analysis{
		Category:       result.Category,
		Priority:       result.Priority,
		Summary:        result.Summary,
		ActionRequired: result.ActionRequired,
		Deadline:       parseDeadline(result.Deadline),
	}

	RefineAnalysis(&analysisResult, subject, sender, body)

	if err := validateAnalysis(analysisResult.Category, analysisResult.Priority); err != nil {
		return Analysis{}, err
	}

	return analysisResult, nil
}

func parseDeadline(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil
	}

	return &t
}

func validateAnalysis(category, priority string) error {
	validCategories := map[string]bool{
		"LEO":       true,
		"IEEE":      true,
		"UNI":       true,
		"JOB":       true,
		"SECURITY":  true,
		"FINANCE":   true,
		"WORK":      true,
		"PERSONAL":  true,
		"PROMOTION": true,
		"SOCIAL":    true,
		"OTHER":     true,
	}

	validPriorities := map[string]bool{
		"CRITICAL": true,
		"HIGH":     true,
		"MEDIUM":   true,
		"LOW":      true,
	}

	if !validCategories[category] {
		return fmt.Errorf("invalid AI category: %q", category)
	}

	if !validPriorities[priority] {
		return fmt.Errorf("invalid AI priority: %q", priority)
	}

	return nil
}

func (a *OpenAIAnalyzer) DraftReply(subject, sender, body, tone string) (string, error) {
	return "Thank you for reaching out. I have received your email and will get back to you shortly.", nil
}

