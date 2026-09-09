package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/Mesit-Rathnayake/mailmind/internal/database"
)

type SyncRunner interface {
	SyncOnce(ctx context.Context)
}

type Server struct {
	db     *database.DB
	worker SyncRunner
}

func NewServer(db *database.DB, worker SyncRunner) *Server {
	return &Server{db: db, worker: worker}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
	mux.HandleFunc("/api/preferences", s.handleGetPreferences)
	return s.corsMiddleware(mux)
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	if s.worker != nil {
		go s.worker.SyncOnce(context.Background())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "sync_triggered",
		"message": "Email ingestion and triage started in background",
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

	ranked, err := s.db.GetRankedEmails(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ranked); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
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
