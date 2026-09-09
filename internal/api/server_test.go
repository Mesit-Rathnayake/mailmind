package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/joho/godotenv"
)

func TestGetPriorityEmailsEndpoint(t *testing.T) {
	_ = godotenv.Load("../../.env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	db, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close(ctx)

	server := NewServer(db, nil)
	handler := server.Routes()

	req := httptest.NewRequest(http.MethodGet, "/api/emails/priority", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	t.Logf("Response body sample: %s", rec.Body.String()[:min(len(rec.Body.String()), 200)])
}
