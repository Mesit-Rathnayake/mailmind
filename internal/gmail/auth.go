package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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
	token, err := loadToken()
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

func cleanJSONEnv(val string) []byte {
	s := strings.TrimSpace(val)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		s = strings.Trim(s, "'")
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		var unquoted string
		if err := json.Unmarshal([]byte(s), &unquoted); err == nil {
			s = unquoted
		}
	}
	return []byte(strings.TrimSpace(s))
}

func loadToken() (*oauth2.Token, error) {
	if raw := os.Getenv("GMAIL_TOKEN_JSON"); raw != "" {
		cleaned := cleanJSONEnv(raw)
		token := &oauth2.Token{}
		if err := json.Unmarshal(cleaned, token); err != nil {
			return nil, fmt.Errorf("failed to parse GMAIL_TOKEN_JSON: %w", err)
		}
		return token, nil
	}
	return tokenFromFile(tokenFile)
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
		log.Printf("Warning: Unable to save token file: %v", err)
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		log.Printf("Warning: Unable to encode token: %v", err)
	}
}

func loadCredentialsBytes() ([]byte, error) {
	if raw := os.Getenv("GMAIL_CREDENTIALS_JSON"); raw != "" {
		cleaned := cleanJSONEnv(raw)
		if len(cleaned) > 0 {
			return cleaned, nil
		}
	}
	return os.ReadFile(credentialsFile)
}

func getClientSafe(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	token, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("OAuth token unavailable: %w", err)
	}
	return config.Client(ctx, token), nil
}

func NewClient(ctx context.Context) *http.Client {
	client, err := NewClientSafe(ctx)
	if err != nil {
		log.Fatalf("Gmail client initialization failed: %v", err)
	}
	return client
}

func NewClientSafe(ctx context.Context) (*http.Client, error) {
	b, err := loadCredentialsBytes()
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials (file %s or GMAIL_CREDENTIALS_JSON env): %w", credentialsFile, err)
	}

	config, err := google.ConfigFromJSON(
		b,
		gmailapi.GmailReadonlyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials JSON: %w", err)
	}

	return getClientSafe(ctx, config)
}

