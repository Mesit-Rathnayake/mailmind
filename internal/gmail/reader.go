package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	stdhtml "html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Mesit-Rathnayake/mailmind/internal/email"
	"golang.org/x/net/html"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var (
	cssBlockRegex     = regexp.MustCompile(`(?is)\{[^}]*\}`)
	atRuleRegex       = regexp.MustCompile(`(?is)@(import|media|keyframes|font-face)[^;{]*\{?[^}]*\}?;?`)
	multiSpaceRegex   = regexp.MustCompile(`[^\S\r\n]+`)
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
)

// cleanEmailText strips leftover CSS artifacts, tags, and cleans whitespace.
func cleanEmailText(text string) string {
	if text == "" {
		return ""
	}

	// Unescape HTML entities (e.g. &nbsp;, &#39;, &amp;)
	text = stdhtml.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")

	// Strip @import, @media rules
	text = atRuleRegex.ReplaceAllString(text, " ")

	// Strip raw CSS declaration blocks { ... }
	text = cssBlockRegex.ReplaceAllString(text, " ")

	// Process line by line
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(multiSpaceRegex.ReplaceAllString(line, " "))
		if trimmed == "" {
			if len(cleanedLines) > 0 && cleanedLines[len(cleanedLines)-1] != "" {
				cleanedLines = append(cleanedLines, "")
			}
			continue
		}

		// Filter out stray CSS selector lines
		if strings.HasPrefix(trimmed, "#outlook") ||
			strings.HasPrefix(trimmed, "@media") ||
			strings.HasPrefix(trimmed, "@import") ||
			strings.HasPrefix(trimmed, ".mj-") ||
			strings.HasPrefix(trimmed, "table.mj-") ||
			strings.HasPrefix(trimmed, "<!--") ||
			strings.HasSuffix(trimmed, "-->") ||
			strings.Contains(trimmed, "mso-table-") ||
			strings.Contains(trimmed, "-webkit-text-size-adjust") {
			continue
		}

		cleanedLines = append(cleanedLines, trimmed)
	}

	result := strings.Join(cleanedLines, "\n")
	result = multiNewlineRegex.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}

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

// htmlToText converts HTML content into readable plain text with structure and line breaks preserved.
func htmlToText(content string) string {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return cleanEmailText(content)
	}

	var builder strings.Builder

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			// Ignore style, script, head, etc. completely
			switch tag {
			case "style", "script", "head", "title", "noscript", "svg", "template", "xml":
				return
			case "br", "hr":
				builder.WriteString("\n")
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				builder.WriteString("\n\n")
			case "li":
				builder.WriteString("\n• ")
			case "tr":
				builder.WriteString("\n")
			case "td", "th":
				builder.WriteString(" ")
			}
		}

		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}

		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			switch tag {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				builder.WriteString("\n")
			}
		}
	}

	walk(doc)

	return cleanEmailText(builder.String())
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
		cleaned := cleanEmailText(plainText)
		if cleaned != "" {
			return cleaned, nil
		}
	}

	// Fall back to HTML if no plain-text version exists.
	if strings.TrimSpace(htmlText) != "" {
		return htmlToText(htmlText), nil
	}

	return "", nil
}

func FetchLatestEmails(ctx context.Context, client *http.Client, count int64) ([]email.Email, error) {
	return FetchLatestEmailsWithCache(ctx, client, count, nil)
}

func FetchLatestEmailsWithCache(ctx context.Context, client *http.Client, count int64, knownIDs map[string]bool) ([]email.Email, error) {
	service, err := gmailapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	user := "me"

	list, err := service.Users.Messages.List(user).
		LabelIds("INBOX").
		Q("in:inbox -in:sent -in:drafts -in:trash").
		MaxResults(count).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	emails := make([]email.Email, 0, len(list.Messages))

	for _, message := range list.Messages {
		// Skip downloading if we already have this email cached/persisted
		if knownIDs != nil && knownIDs[message.Id] {
			continue
		}

		// Request the complete Gmail message
		msg, err := service.Users.Messages.Get(user, message.Id).
			Format("full").
			Do()
		if err != nil {
			continue
		}

		// Verify it's not a sent message
		isSent := false
		isRead := true
		isStarred := false
		for _, label := range msg.LabelIds {
			if label == "SENT" || label == "DRAFT" || label == "TRASH" {
				isSent = true
				break
			}
			if label == "UNREAD" {
				isRead = false
			}
			if label == "STARRED" {
				isStarred = true
			}
		}
		if isSent {
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

		// Use Gmail InternalDate (milliseconds epoch) for exact received timestamp
		var parsedDate time.Time
		if msg.InternalDate > 0 {
			parsedDate = time.UnixMilli(msg.InternalDate)
		} else {
			parsedDate = parseEmailDate(date)
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
