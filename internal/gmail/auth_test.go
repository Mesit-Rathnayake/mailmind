package gmail

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestShouldRefreshToken(t *testing.T) {
	t.Run("nil token needs refresh", func(t *testing.T) {
		if !shouldRefreshToken(nil) {
			t.Fatal("nil token should require refresh")
		}
	})

	t.Run("expired token needs refresh", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken: "expired",
			Expiry:      time.Now().Add(-1 * time.Hour),
		}
		if !shouldRefreshToken(token) {
			t.Fatal("expired token should require refresh")
		}
	})

	t.Run("expired token with refresh token can be refreshed", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken:  "expired",
			RefreshToken: "refresh",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		if shouldRefreshToken(token) {
			t.Fatal("expired token with a refresh token should use token refresh")
		}
	})

	t.Run("valid token with refresh token does not need refresh", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken:  "valid",
			RefreshToken: "refresh",
			Expiry:       time.Now().Add(1 * time.Hour),
		}
		if shouldRefreshToken(token) {
			t.Fatal("valid token should not require refresh")
		}
	})
}
