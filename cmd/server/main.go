package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/api"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
	"github.com/Mesit-Rathnayake/mailmind/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found or failed to load")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := database.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close(ctx)

	log.Println("Database connection established!")

	server := api.NewServer(db)

	// Start Background Ingestion & Triage Worker
	syncInterval := 5 * time.Minute
	if intervalStr := os.Getenv("SYNC_INTERVAL_MINUTES"); intervalStr != "" {
		if mins, err := strconv.Atoi(intervalStr); err == nil && mins > 0 {
			syncInterval = time.Duration(mins) * time.Minute
		}
	}

	w, err := worker.NewWorker(db, syncInterval)
	if err != nil {
		log.Printf("Worker setup warning: %v", err)
	} else {
		w.Start(ctx)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("MailMind API server listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
