package mailbox

import "time"

// Message is a provider-agnostic inbound email.
type Message struct {
	UID       string    // provider-stable unique id (for Mark)
	ThreadID  string    // conversation id; may equal UID if unsupported
	From      string    // sender address
	To        []string  // recipients
	Subject   string
	Body      string    // plain-text body
	MessageID string    // RFC 5322 Message-ID header (for threading replies)
	InReplyTo string
	Date      time.Time
}

// OutMessage is a provider-agnostic outbound email.
type OutMessage struct {
	To        string
	Subject   string
	Body      string            // plain-text body
	InReplyTo string            // Message-ID to thread under (replies)
	Headers   map[string]string // extra headers, e.g. List-Unsubscribe
}

// Mailbox abstracts a service mailbox the user owns. Implementations:
//   - imap.go:  IMAP (fetch) + SMTP (send), any provider
//   - gmail.go: Gmail API (OAuth)
//
// Implementations MUST represent one real, owned address; they never create,
// spoof, or rotate identities.
type Mailbox interface {
	// Fetch returns up to `limit` unprocessed inbound messages.
	Fetch(limit int) ([]Message, error)
	// Send delivers an outbound message from the owned address.
	Send(msg OutMessage) error
	// Mark flags a message as processed so it is not fetched again.
	Mark(uid string) error
	// Close releases any open connections.
	Close() error
}
