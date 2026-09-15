package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/email"
	"golang.org/x/net/html"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func parseEmailDate(value string) time.Time {
	// Remove optional timezone name, e.g. "(UTC)", "(IST)", "(CST)".
	if index := strings.LastIndex(value, " ("); index != -1 {
		value = value[:index]
	}

	// Gmail can return either "3 Sep" or "03 Sep".
	// Normalize a single-digit day to two digits.
	parts := strings.Fields(value)

	if len(parts) >= 2 && len(parts[1]) == 1 {
		parts[1] = "0" + parts[1]
		value = strings.Join(parts, " ")
	}

	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		time.RFC3339,
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// decodeBody decodes Gmail's URL-safe base64 encoded message body.
func decodeBody(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		// Some Gmail responses may contain padding.
		decoded, err = base64.URLEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("failed to decode email body: %w", err)
		}
	}

	return string(decoded), nil
}

// htmlToText converts HTML content into readable plain text.
func htmlToText(content string) string {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return content
	}

	var builder strings.Builder

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
			builder.WriteString(" ")
			return
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(doc)

	return strings.Join(strings.Fields(builder.String()), " ")
}

// extractBody recursively walks Gmail's MIME structure.
func extractBody(payload *gmailapi.MessagePart) (string, error) {
	if payload == nil {
		return "", nil
	}

	var plainText string
	var htmlText string

	// Check this part's body.
	if payload.Body != nil && payload.Body.Data != "" {
		body, err := decodeBody(payload.Body.Data)
		if err != nil {
			return "", err
		}

		switch strings.ToLower(payload.MimeType) {
		case "text/plain":
			plainText = body
		case "text/html":
			htmlText = body
		}
	}

	// Recursively inspect child parts.
	for _, part := range payload.Parts {
		body, err := extractBody(part)
		if err != nil {
			return "", err
		}

		if body == "" {
			continue
		}

		switch strings.ToLower(part.MimeType) {
		case "text/plain":
			if plainText == "" {
				plainText = body
			}
		case "text/html":
			if htmlText == "" {
				htmlText = body
			}
		default:
			if plainText == "" {
				plainText = body
			}
		}
	}

	// Prefer plain text because it is better for AI processing.
	if strings.TrimSpace(plainText) != "" {
		return strings.TrimSpace(plainText), nil
	}

	// Fall back to HTML if no plain-text version exists.
	if strings.TrimSpace(htmlText) != "" {
		return strings.TrimSpace(htmlToText(htmlText)), nil
	}

	return "", nil
}

func FetchLatestEmails(ctx context.Context, client *http.Client, count int64) ([]email.Email, error) {
	service, err := gmailapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	user := "me"

	list, err := service.Users.Messages.List(user).
		MaxResults(count).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	emails := make([]email.Email, 0, len(list.Messages))

	for _, message := range list.Messages {
		// Request the complete Gmail message instead of metadata only.
		msg, err := service.Users.Messages.Get(user, message.Id).
			Format("full").
			Do()
		if err != nil {
			continue
		}

		var from, to, subject, date string

		for _, header := range msg.Payload.Headers {
			switch header.Name {
			case "From":
				from = header.Value
			case "To":
				to = header.Value
			case "Subject":
				subject = header.Value
			case "Date":
				date = header.Value
			}
		}

		body, err := extractBody(msg.Payload)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to extract body from email %s: %w",
				msg.Id,
				err,
			)
		}

		parsedDate := parseEmailDate(date)

		isRead := true
		isStarred := false
		for _, label := range msg.LabelIds {
			if label == "UNREAD" {
				isRead = false
			}
			if label == "STARRED" {
				isStarred = true
			}
		}

		emails = append(emails, email.Email{
			ID:        msg.Id,
			ThreadID:  msg.ThreadId,
			From:      from,
			To:        to,
			Subject:   subject,
			Date:      parsedDate,
			Snippet:   msg.Snippet,
			Body:      body,
			IsRead:    isRead,
			IsStarred: isStarred,
			Labels:    msg.LabelIds,
		})
	}

	return emails, nil
}
