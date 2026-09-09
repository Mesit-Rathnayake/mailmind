package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Mesit-Rathnayake/mailmind/internal/api"
	"github.com/Mesit-Rathnayake/mailmind/internal/database"
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
