package ai

import (
	"context"
	"fmt"
	"log"
)

func RunOpenAITest() {
	analyzer, err := NewGeminiAnalyzer(context.Background())
	if err != nil {
		log.Fatalf("Failed to create OpenAI analyzer: %v", err)
	}

	result, err := analyzer.Analyze(
		"Interview scheduled for Friday",
		"hr@example.com",
		"Hi, your technical interview has been scheduled for Friday at 10:00 AM. Please confirm your attendance before Thursday.",
	)
	if err != nil {
		log.Fatalf("AI analysis failed: %v", err)
	}

	fmt.Println("\n========== AI TEST ==========")
	fmt.Printf("Category:        %s\n", result.Category)
	fmt.Printf("Priority:        %s\n", result.Priority)
	fmt.Printf("Summary:         %s\n", result.Summary)
	fmt.Printf("Action Required: %t\n", result.ActionRequired)

	if result.Deadline != nil {
		fmt.Printf("Deadline:        %s\n", result.Deadline.Format("2006-01-02 15:04:05 MST"))
	} else {
		fmt.Println("Deadline:        none")
	}
}
