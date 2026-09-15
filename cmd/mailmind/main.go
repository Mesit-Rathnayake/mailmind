package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/ai"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/Mesit-Rathnayake/mailmind/internal/gmail"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting MailMind...")

	if err := godotenv.Load(); err != nil {
		log.Fatal("Failed to load .env")
	}

	ctx := context.Background()

	// -------------------------
	// Database
	// -------------------------

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := database.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close(ctx)

	log.Println("PostgreSQL connection successful!")

	// -------------------------
	// Gmail
	// -------------------------

	client := gmail.NewClient(ctx)

	log.Println("Gmail authentication successful!")

	emails, err := gmail.FetchLatestEmails(ctx, client, 10)
	if err != nil {
		log.Fatalf("Failed to fetch emails: %v", err)
	}

	log.Printf("Fetched %d emails", len(emails))

	// Save emails to database.
	for _, e := range emails {
		if err := db.SaveEmail(ctx, e); err != nil {
			log.Printf("Failed to save email %s: %v", e.ID, err)
			continue
		}
	}

	// -------------------------
	// AI Analyzer
	// -------------------------

	analyzer, err := ai.NewAnalyzerFromEnv(ctx)
	if err != nil {
		log.Fatalf("Failed to create AI analyzer: %v", err)
	}

	log.Printf("AI analyzer initialized for provider: %s", strings.TrimSpace(strings.ToLower(os.Getenv("AI_PROVIDER"))))

	// -------------------------
	// User Preferences
	// -------------------------

	prefs, err := db.GetUserPreferences(ctx)
	if err != nil {
		log.Fatalf("Failed to get user preferences: %v", err)
	}

	log.Printf("Loaded %d user preference rules", len(prefs))

	// -------------------------
	// AI Processing
	// -------------------------

	unprocessed, err := db.GetUnprocessedEmails(ctx, 5)
	if err != nil {
		log.Fatalf("Failed to get unprocessed emails: %v", err)
	}

	log.Printf("Found %d unprocessed emails", len(unprocessed))

	for i, e := range unprocessed {
		log.Printf(
			"Analyzing email %d/%d: %s",
			i+1,
			len(unprocessed),
			e.Subject,
		)

		analysis, err := analyzer.Analyze(
			e.Subject,
			e.From,
			e.Body,
		)

		if err != nil {
			log.Printf(
				"AI analysis failed for email %s: %v",
				e.ID,
				err,
			)
			continue
		}

		analysis.AttentionScore = ai.CalculateAttentionScore(e, analysis, prefs)

		err = db.SaveAnalysis(
			ctx,
			e.ID,
			analysis.Category,
			analysis.Priority,
			analysis.Summary,
			analysis.ActionRequired,
			analysis.Deadline,
			analysis.AttentionScore,
		)

		if err != nil {
			log.Printf(
				"Failed to save analysis for email %s: %v",
				e.ID,
				err,
			)
			continue
		}

		log.Printf(
			"Analyzed: [%s] [%s] [Score: %d] %s",
			analysis.Category,
			analysis.Priority,
			analysis.AttentionScore,
			analysis.Summary,
		)

		time.Sleep(1 * time.Second)
	}

	log.Println("MailMind processing complete!")
}
