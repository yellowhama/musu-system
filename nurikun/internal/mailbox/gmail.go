package mailbox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// gmailUser is the special "me" alias the Gmail API uses for the authenticated
// user's own mailbox.
const gmailUser = "me"

// gmailMailbox is a Mailbox backed by the Gmail API (OAuth2).
//
// It represents the single, real, owned account authorized by the cached
// token; it never spoofs or rotates identities.
type gmailMailbox struct {
	svc  *gmail.Service
	from string
}

// newGmailMailbox builds a Gmail-backed mailbox. It loads the OAuth client
// credentials from s.GmailCredentials and a previously-cached token from
// s.GmailToken, then constructs an authorized gmail.Service.
//
// The token must already be provisioned (see Close docs / package README): the
// agent performs no interactive OAuth flow at runtime.
func newGmailMailbox(s Settings) (*gmailMailbox, error) {
	if s.GmailCredentials == "" {
		return nil, fmt.Errorf("gmail mailbox: GmailCredentials path is required")
	}
	if s.GmailToken == "" {
		return nil, fmt.Errorf("gmail mailbox: GmailToken path is required")
	}

	credBytes, err := os.ReadFile(s.GmailCredentials)
	if err != nil {
		return nil, fmt.Errorf("gmail mailbox: read credentials %q: %w", s.GmailCredentials, err)
	}

	// gmail.MailGoogleComScope grants read, send and modify on the user's own
	// mailbox, covering Fetch/Send/Mark.
	config, err := google.ConfigFromJSON(credBytes,
		gmail.GmailReadonlyScope,
		gmail.GmailSendScope,
		gmail.GmailModifyScope,
	)
	if err != nil {
		return nil, fmt.Errorf("gmail mailbox: parse credentials: %w", err)
	}

	token, err := loadGmailToken(s.GmailToken)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	// TokenSource auto-refreshes the access token using the refresh token.
	httpClient := config.Client(ctx, token)

	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("gmail mailbox: create service: %w", err)
	}

	from := s.From
	if from == "" {
		from = gmailUser
	}

	return &gmailMailbox{svc: svc, from: from}, nil
}

// loadGmailToken reads a cached oauth2 token (JSON) from disk.
func loadGmailToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gmail mailbox: open token %q: %w", path, err)
	}
	defer f.Close()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("gmail mailbox: decode token %q: %w", path, err)
	}
	return tok, nil
}

// Fetch returns up to `limit` unread INBOX messages.
func (m *gmailMailbox) Fetch(limit int) ([]Message, error) {
	if limit <= 0 {
		return nil, nil
	}

	listResp, err := m.svc.Users.Messages.List(gmailUser).
		LabelIds("UNREAD", "INBOX").
		MaxResults(int64(limit)).
		Do()
	if err != nil {
		return nil, fmt.Errorf("gmail mailbox: list messages: %w", err)
	}

	messages := make([]Message, 0, len(listResp.Messages))
	for _, ref := range listResp.Messages {
		full, err := m.svc.Users.Messages.Get(gmailUser, ref.Id).Format("full").Do()
		if err != nil {
			return nil, fmt.Errorf("gmail mailbox: get message %s: %w", ref.Id, err)
		}
		messages = append(messages, gmailToMessage(full))
	}
	return messages, nil
}

// gmailToMessage converts a Gmail API message into a provider-agnostic Message.
func gmailToMessage(g *gmail.Message) Message {
	msg := Message{
		UID:      g.Id,
		ThreadID: g.ThreadId,
	}
	if g.InternalDate > 0 {
		msg.Date = time.UnixMilli(g.InternalDate)
	}

	if g.Payload != nil {
		for _, h := range g.Payload.Headers {
			switch strings.ToLower(h.Name) {
			case "from":
				msg.From = h.Value
			case "to":
				msg.To = splitAddressList(h.Value)
			case "subject":
				msg.Subject = h.Value
			case "message-id":
				msg.MessageID = strings.Trim(h.Value, "<>")
			case "in-reply-to":
				msg.InReplyTo = strings.Trim(h.Value, "<>")
			case "date":
				if msg.Date.IsZero() {
					if t, err := parseMailDate(h.Value); err == nil {
						msg.Date = t
					}
				}
			}
		}
		msg.Body = extractGmailPlainText(g.Payload)
	}
	return msg
}

// extractGmailPlainText walks the MIME tree of a Gmail payload and returns the
// first text/plain part, falling back to the top-level body if none is found.
func extractGmailPlainText(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(part.MimeType), "text/plain") {
		if decoded := decodeGmailBody(part.Body); decoded != "" {
			return decoded
		}
	}

	for _, child := range part.Parts {
		if text := extractGmailPlainText(child); text != "" {
			return text
		}
	}

	// Fallback: a single-part message with no explicit text/plain MIME type.
	if len(part.Parts) == 0 && !strings.HasPrefix(strings.ToLower(part.MimeType), "text/html") {
		return decodeGmailBody(part.Body)
	}
	return ""
}

// decodeGmailBody base64url-decodes a Gmail message part body.
func decodeGmailBody(body *gmail.MessagePartBody) string {
	if body == nil || body.Data == "" {
		return ""
	}
	decoded, err := base64.URLEncoding.DecodeString(body.Data)
	if err != nil {
		// Gmail uses URL-safe base64; some bodies omit padding.
		decoded, err = base64.RawURLEncoding.DecodeString(body.Data)
		if err != nil {
			return ""
		}
	}
	return string(decoded)
}

// Send delivers an outbound message from the owned account via the Gmail API.
func (m *gmailMailbox) Send(out OutMessage) error {
	if out.To == "" {
		return fmt.Errorf("gmail mailbox: outbound message has no recipient")
	}

	fromHeader := m.from
	if fromHeader == gmailUser {
		fromHeader = ""
	}
	raw := buildRFC5322(fromHeader, out)

	gmsg := &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString(raw),
	}
	if _, err := m.svc.Users.Messages.Send(gmailUser, gmsg).Do(); err != nil {
		return fmt.Errorf("gmail mailbox: send message: %w", err)
	}
	return nil
}

// Mark removes the UNREAD label so the message is not fetched again.
func (m *gmailMailbox) Mark(uid string) error {
	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}
	if _, err := m.svc.Users.Messages.Modify(gmailUser, uid, req).Do(); err != nil {
		return fmt.Errorf("gmail mailbox: mark message %s read: %w", uid, err)
	}
	return nil
}

// Close is a no-op: the Gmail API client holds no long-lived connection that
// needs explicit release.
func (m *gmailMailbox) Close() error {
	return nil
}

// splitAddressList splits a comma-separated header value into trimmed addresses.
func splitAddressList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseMailDate parses common RFC5322 Date header formats.
func parseMailDate(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, "Mon, 2 Jan 2006 15:04:05 -0700", "2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date %q", v)
}
