package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

type GeminiAnalyzer struct {
	client *genai.Client
	model  string
}

func NewGeminiAnalyzer(ctx context.Context) (*GeminiAnalyzer, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-3.7-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiAnalyzer{
		client: client,
		model:  model,
	}, nil
}

func (a *GeminiAnalyzer) Model() string {
	return a.model
}

func (a *GeminiAnalyzer) Analyze(
	subject string,
	sender string,
	body string,
) (Analysis, error) {

	ctx := context.Background()

	systemInstruction := `
You are MailMind, an intelligent email triage assistant.

Analyze the email and classify it.

Allowed categories:
LEO, IEEE, UNI, JOB, SECURITY, FINANCE, WORK, PERSONAL, PROMOTION, SOCIAL, OTHER

Category definitions:
- LEO: Leo Club, Leo District (e.g. 306), Lions/Leo meetings, installation ceremonies, invitations, notices, leo portal. (ALWAYS use LEO if it mentions Leo/Lions, NEVER PERSONAL or WORK).
- IEEE: IEEE Student Branch, IEEE memberships/renewals, conferences, hackathons, competitions, tech events, and webinars.
- UNI: University announcements, Faculty of Engineering notices, lecturers, coursework, exams, academic alerts.
- JOB: Internship opportunities, job offers, LinkedIn job alerts, interview requests, hiring notices.
- SECURITY: Security alerts, password resets, verification codes, 2FA notifications, sign-in alerts.
- FINANCE: Banking, statements, receipts, invoices, subscription payments, billing.
- WORK: Professional workplace tasks, direct team/client project assignments only. (NEVER classify social media or Reddit notifications as WORK).
- PERSONAL: Direct personal emails from friends or family not related to clubs or automated services.
- PROMOTION: Marketing blasts, product discounts, newsletter promotions, deals.
- SOCIAL: Reddit (r/...), Twitter/X, Instagram, Facebook, YouTube, Discord, Quora, Medium, LinkedIn reactions/connections. (ALWAYS use SOCIAL for Reddit, NEVER WORK).
- OTHER: Any other emails that do not fit into the categories above.

Allowed priorities:
CRITICAL, HIGH, MEDIUM, LOW

Rules:
- category must be exactly one allowed category.
- priority must be exactly one allowed priority.
- summary must be concise and factual.
- action_required is true only when the recipient needs to do something.
- deadline must be an RFC3339 timestamp when a meaningful deadline is explicitly mentioned.
- deadline must be null when there is no meaningful deadline.
- Never invent a deadline.
`

	prompt := fmt.Sprintf(
		`Analyze this email.

Sender: %s
Subject: %s

Email body:
%s`,
		sender,
		subject,
		body,
	)

	response, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(
				systemInstruction,
				genai.RoleUser,
			),
			Temperature:      genai.Ptr(float32(0.1)),
			MaxOutputTokens:  500,
			ResponseMIMEType: "application/json",
			ResponseJsonSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type": "string",
						"enum": []string{
							"LEO",
							"IEEE",
							"UNI",
							"JOB",
							"SECURITY",
							"FINANCE",
							"WORK",
							"PERSONAL",
							"PROMOTION",
							"SOCIAL",
							"OTHER",
						},
					},
					"priority": map[string]any{
						"type": "string",
						"enum": []string{
							"CRITICAL",
							"HIGH",
							"MEDIUM",
							"LOW",
						},
					},
					"summary": map[string]any{
						"type": "string",
					},
					"action_required": map[string]any{
						"type": "boolean",
					},
					"deadline": map[string]any{
						"anyOf": []any{
							map[string]any{
								"type":   "string",
								"format": "date-time",
							},
							map[string]any{
								"type": "null",
							},
						},
					},
				},
				"required": []string{
					"category",
					"priority",
					"summary",
					"action_required",
					"deadline",
				},
				"additionalProperties": false,
			},
		},
	)

	if err != nil {
		return Analysis{}, fmt.Errorf("Gemini request failed: %w", err)
	}

	if response == nil || len(response.Candidates) == 0 {
		return Analysis{}, fmt.Errorf("Gemini returned no candidates")
	}

	output := strings.TrimSpace(response.Text())

	if output == "" {
		return Analysis{}, fmt.Errorf("Gemini returned an empty response")
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
			"failed to parse Gemini response: %w\nResponse: %s",
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

func (a *GeminiAnalyzer) DraftReply(subject, sender, body, tone string) (string, error) {
	if tone == "" {
		tone = "professional, concise, and polite"
	}

	cleanBody := strings.TrimSpace(body)
	if len(cleanBody) > 3000 {
		cleanBody = cleanBody[:3000] + "..."
	}

	prompt := fmt.Sprintf(`Draft a direct email reply to this email.
Tone: %s
Do not include subject lines or metadata markers. Provide only the email body response.

Sender: %s
Subject: %s
Original Email:
%s`, tone, sender, subject, cleanBody)

	ctx := context.Background()
	response, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.3)),
		},
	)
	if err != nil {
		return "", fmt.Errorf("gemini draft failed: %w", err)
	}

	return strings.TrimSpace(response.Text()), nil
}

