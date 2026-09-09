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
WORK, FINANCE, PERSONAL, PROMOTION, SOCIAL, JOB, SECURITY, OTHER

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

	if err := validateAnalysis(result.Category, result.Priority); err != nil {
		return Analysis{}, err
	}

	return Analysis{
		Category:       result.Category,
		Priority:       result.Priority,
		Summary:        result.Summary,
		ActionRequired: result.ActionRequired,
		Deadline:       parseDeadline(result.Deadline),
	}, nil
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
