package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// Load database connection string
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Connect to PostgreSQL
	db, err := database.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close(ctx)

	log.Println("PostgreSQL connection successful!")

	// Connect to Gmail
	client := gmail.NewClient(ctx)

	log.Println("Gmail authentication successful!")

	emails, err := gmail.FetchLatestEmails(ctx, client, 10)
	if err != nil {
		log.Fatalf("Failed to fetch emails: %v", err)
	}

	log.Printf("Fetched %d emails", len(emails))

	for _, email := range emails {
		if err := db.SaveEmail(ctx, email); err != nil {
			log.Printf("Failed to save email %s: %v", email.ID, err)
			continue
		}

		log.Printf("Saved email: %s", email.Subject)
	}

	for i, email := range emails {
		fmt.Printf("\n========== EMAIL %d ==========\n", i+1)
		fmt.Printf("ID: %s\n", email.ID)
		fmt.Printf("From: %s\n", email.From)
		fmt.Printf("Subject: %s\n", email.Subject)
		fmt.Printf("Date: %s\n", email.Date)
		fmt.Printf("Snippet: %s\n", email.Snippet)
		fmt.Printf("Body length: %d characters\n", len(email.Body))
	}
}
