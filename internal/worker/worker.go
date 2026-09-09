package worker

import (
	"context"
	"log"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/ai"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/Mesit-Rathnayake/mailmind/internal/gmail"
)

type Worker struct {
	db       *database.DB
	analyzer ai.Analyzer
	interval time.Duration
}

func NewWorker(db *database.DB, interval time.Duration) (*Worker, error) {
	ctx := context.Background()
	analyzer, err := ai.NewGeminiAnalyzer(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create Gemini analyzer for worker: %v", err)
	}

	return &Worker{
		db:       db,
		analyzer: analyzer,
		interval: interval,
	}, nil
}

func (w *Worker) Start(ctx context.Context) {
	log.Printf("Starting background email sync worker (interval: %v)...", w.interval)

	// Run initial sync shortly after boot
	go func() {
		time.Sleep(5 * time.Second)
		w.SyncOnce(ctx)
	}()

	ticker := time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Println("Background sync worker stopped.")
				return
			case <-ticker.C:
				w.SyncOnce(ctx)
			}
		}
	}()
}

func (w *Worker) SyncOnce(ctx context.Context) {
	log.Println("[Worker] Running scheduled email sync & triage...")

	client, err := gmail.NewClientSafe(ctx)
	if err != nil {
		log.Printf("[Worker] Skipping sync - Gmail credentials unavailable: %v", err)
		return
	}

	// 1. Fetch latest emails from Gmail
	emails, err := gmail.FetchLatestEmails(ctx, client, 15)
	if err != nil {
		log.Printf("[Worker] Error fetching emails from Gmail: %v", err)
		return
	}

	for _, e := range emails {
		if err := w.db.SaveEmail(ctx, e); err != nil {
			log.Printf("[Worker] Failed to save email %s: %v", e.ID, err)
		}
	}

	// 2. Load user preferences for scoring
	prefs, err := w.db.GetUserPreferences(ctx)
	if err != nil {
		log.Printf("[Worker] Failed to load preferences: %v", err)
	}

	// 3. Process unprocessed emails with Gemini
	if w.analyzer == nil {
		w.analyzer, err = ai.NewGeminiAnalyzer(ctx)
		if err != nil {
			log.Printf("[Worker] Gemini analyzer unavailable: %v", err)
			return
		}
	}

	unprocessed, err := w.db.GetUnprocessedEmails(ctx, 10)
	if err != nil {
		log.Printf("[Worker] Failed to get unprocessed emails: %v", err)
		return
	}

	if len(unprocessed) == 0 {
		log.Println("[Worker] Pipeline up-to-date. No unprocessed emails.")
		return
	}

	log.Printf("[Worker] Processing %d unanalyzed emails with AI...", len(unprocessed))

	for _, e := range unprocessed {
		analysis, err := w.analyzer.Analyze(e.Subject, e.From, e.Body)
		if err != nil {
			log.Printf("[Worker] AI analysis failed for email %s: %v", e.ID, err)
			continue
		}

		analysis.AttentionScore = ai.CalculateAttentionScore(e, analysis, prefs)

		err = w.db.SaveAnalysis(
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
			log.Printf("[Worker] Failed to save analysis for %s: %v", e.ID, err)
			continue
		}

		log.Printf("[Worker] Analyzed %s -> [%s] [Score: %d]", e.Subject, analysis.Category, analysis.AttentionScore)
		time.Sleep(1 * time.Second) // Rate limiting buffer
	}

	log.Println("[Worker] Email sync & triage complete.")
}
