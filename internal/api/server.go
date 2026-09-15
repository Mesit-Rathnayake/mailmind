package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/ai"
	"github.com/Mesit-Rathnayake/mailmind/internal/cache"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/Mesit-Rathnayake/mailmind/internal/email"
)

type SyncRunner interface {
	SyncOnce(ctx context.Context) (int, error)
}

type Server struct {
	db       *database.DB
	worker   SyncRunner
	analyzer ai.Analyzer
	cache    *cache.MemoryCache
}

func NewServer(db *database.DB, worker SyncRunner) *Server {
	ctx := context.Background()
	analyzer, err := ai.NewAnalyzerFromEnv(ctx)
	if err != nil {
		log.Printf("Warning: API Server failed to initialize AI analyzer: %v", err)
	}

	return &Server{
		db:       db,
		worker:   worker,
		analyzer: analyzer,
		cache:    cache.NewMemoryCache(),
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/sync", s.handleTriggerSync)
	mux.HandleFunc("/api/emails/priority", s.handleGetPriorityEmails)
	mux.HandleFunc("/api/emails/stats", s.handleGetEmailStats)
	mux.HandleFunc("/api/emails/status", s.handleUpdateEmailStatus)
	mux.HandleFunc("/api/emails/generate-reply", s.handleGenerateReply)
	mux.HandleFunc("/api/preferences", s.handleGetPreferences)
	return s.corsMiddleware(mux)
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.worker == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "Worker service not initialized",
		})
		return
	}

	count, err := s.worker.SyncOnce(context.Background())
	if err != nil {
		log.Printf("Sync trigger error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Invalidate cache after sync
	s.cache.Clear()

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"synced": count,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "mailmind-api",
	})
}

func (s *Server) handleGetPriorityEmails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	filter := email.RankedFilter{
		TimeFrame: r.URL.Query().Get("timeframe"),
		Status:    r.URL.Query().Get("status"),
		Category:  r.URL.Query().Get("category"),
		Search:    r.URL.Query().Get("search"),
		Limit:     limit,
	}

	cacheKey := fmt.Sprintf("ranked:%s:%s:%s:%s:%d", filter.TimeFrame, filter.Status, filter.Category, filter.Search, filter.Limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_ = json.NewEncoder(w).Encode(cached)
		return
	}

	ranked, err := s.db.GetRankedEmails(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.cache.Set(cacheKey, ranked, 45*time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(ranked); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleGetEmailStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cacheKey := "stats"
	if cached, ok := s.cache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		_ = json.NewEncoder(w).Encode(cached)
		return
	}

	stats, err := s.db.GetEmailStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.cache.Set(cacheKey, stats, 45*time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode stats: %v", err)
	}
}

func (s *Server) handleUpdateEmailStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update email.EmailStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(update.GmailID) == "" {
		http.Error(w, "Missing gmail_id", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateEmailStatus(r.Context(), update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cache immediately on status change
	s.cache.Clear()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"gmail_id": update.GmailID,
	})
}

func (s *Server) handleGenerateReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GmailID string `json:"gmail_id"`
		Subject string `json:"subject"`
		Sender  string `json:"sender"`
		Body    string `json:"body"`
		Tone    string `json:"tone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if s.analyzer == nil {
		var err error
		s.analyzer, err = ai.NewAnalyzerFromEnv(r.Context())
		if err != nil {
			http.Error(w, "AI analyzer unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}

	draft, err := s.analyzer.DraftReply(req.Subject, req.Sender, req.Body, req.Tone)
	if err != nil {
		log.Printf("Failed to draft reply: %v", err)
		http.Error(w, "Failed to generate reply: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if req.GmailID != "" {
		_ = s.db.UpdateEmailStatus(r.Context(), email.EmailStatusUpdate{
			GmailID:    req.GmailID,
			DraftReply: &draft,
		})
		s.cache.Clear()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"draft":  draft,
	})
}

func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prefs, err := s.db.GetUserPreferences(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prefs); err != nil {
		log.Printf("Failed to encode preferences: %v", err)
	}
}
