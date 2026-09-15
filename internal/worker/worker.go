package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/ai"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/Mesit-Rathnayake/mailmind/internal/gmail"
)

type Worker struct {
	db       *database.DB
	analyzer ai.Analyzer
	interval time.Duration
	syncMu   sync.Mutex
}

func NewWorker(db *database.DB, interval time.Duration) (*Worker, error) {
	ctx := context.Background()
	analyzer, err := ai.NewAnalyzerFromEnv(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create AI analyzer for worker: %v", err)
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
		time.Sleep(3 * time.Second)
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

func (w *Worker) SyncOnce(ctx context.Context) (int, error) {
	w.syncMu.Lock()
	defer w.syncMu.Unlock()

	log.Println("[Worker] Running scheduled email sync & triage...")

	client, err := gmail.NewClientSafe(ctx)
	if err != nil {
		log.Printf("[Worker] Skipping sync - Gmail credentials unavailable: %v", err)
		return 0, fmt.Errorf("gmail client setup failed: %w", err)
	}

	// 1. Determine batch fetch count (default 50)
	fetchCount := int64(50)
	if envFetch := os.Getenv("GMAIL_FETCH_COUNT"); envFetch != "" {
		if c, err := strconv.ParseInt(envFetch, 10, 64); err == nil && c > 0 {
			fetchCount = c
		}
	}

	// 2. Fetch latest emails from Gmail using deduplication cache
	emails, err := gmail.FetchLatestEmailsWithCache(ctx, client, fetchCount, nil)
	if err != nil {
		log.Printf("[Worker] Error fetching emails from Gmail: %v", err)
		return 0, fmt.Errorf("error fetching emails from Gmail: %w", err)
	}

	savedCount := 0
	for _, e := range emails {
		if err := w.db.SaveEmail(ctx, e); err != nil {
			log.Printf("[Worker] Failed to save email %s: %v", e.ID, err)
		} else {
			savedCount++
		}
	}
	log.Printf("[Worker] Synced %d emails from Gmail (saved/updated: %d)", len(emails), savedCount)

	// 3. Load user preferences for scoring
	prefs, err := w.db.GetUserPreferences(ctx)
	if err != nil {
		log.Printf("[Worker] Failed to load preferences: %v", err)
	}

	// 4. Process unprocessed emails with AI
	if w.analyzer == nil {
		w.analyzer, err = ai.NewAnalyzerFromEnv(ctx)
		if err != nil {
			log.Printf("[Worker] AI analyzer unavailable: %v", err)
			return savedCount, fmt.Errorf("ai analyzer unavailable: %w", err)
		}
	}

	aiBatchSize := 20
	if envAi := os.Getenv("AI_BATCH_SIZE"); envAi != "" {
		if b, err := strconv.Atoi(envAi); err == nil && b > 0 {
			aiBatchSize = b
		}
	}

	unprocessed, err := w.db.GetUnprocessedEmails(ctx, aiBatchSize)
	if err != nil {
		log.Printf("[Worker] Failed to get unprocessed emails: %v", err)
		return savedCount, err
	}

	if len(unprocessed) == 0 {
		log.Println("[Worker] Pipeline up-to-date. No unprocessed emails.")
		return savedCount, nil
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
		time.Sleep(300 * time.Millisecond) // Smooth rate limiting buffer
	}

	log.Println("[Worker] Email sync & triage complete.")
	return savedCount, nil
}
