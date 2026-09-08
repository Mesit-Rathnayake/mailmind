package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"
)

const (
	credentialsFile = "credentials.json"
	tokenFile       = "token.json"
	redirectURL     = "http://localhost:8080/oauth2callback"
)

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	token, err := tokenFromFile(tokenFile)
	if err != nil {
		token = getTokenFromWeb(ctx, config)
		saveToken(tokenFile, token)
	}

	return config.Client(ctx, token)
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
	)

	codeChan := make(chan string)
	errorChan := make(chan error)

	server := &http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errorChan <- fmt.Errorf("OAuth error: %s", errMsg)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errorChan <- fmt.Errorf("authorization code not found")
			return
		}

		fmt.Fprintln(w, "MailMind authentication successful! You can close this browser tab.")

		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	fmt.Println("Opening Google authorization URL:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Open the URL above in your browser if it did not open automatically.")

	if err := openBrowser(authURL); err != nil {
		fmt.Println("Could not open browser automatically.")
	}

	var code string

	select {
	case code = <-codeChan:
	case err := <-errorChan:
		log.Fatalf("OAuth authentication failed: %v", err)
	}

	server.Shutdown(ctx)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Unable to exchange authorization code: %v", err)
	}

	return token
}

func openBrowser(targetURL string) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("unexpected URL scheme: %s", parsedURL.Scheme)
	}

	// Windows
	cmd := fmt.Sprintf("start \"\" \"%s\"", targetURL)

	return (&commandRunner{}).Run(cmd)
}

type commandRunner struct{}

func (c *commandRunner) Run(command string) error {
	return nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}

	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)

	f, err := os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil {
		log.Fatalf("Unable to save token: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		log.Fatalf("Unable to encode token: %v", err)
	}
}

func NewClient(ctx context.Context) *http.Client {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read credentials.json: %v", err)
	}

	config, err := google.ConfigFromJSON(
		b,
		gmailapi.GmailReadonlyScope,
	)
	if err != nil {
		log.Fatalf("Unable to parse credentials.json: %v", err)
	}

	return getClient(ctx, config)
}
