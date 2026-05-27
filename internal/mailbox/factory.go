package mailbox

import "fmt"

// Settings carries the connection parameters a Mailbox implementation needs.
// The caller populates it from config.
type Settings struct {
	// IMAP (fetch) + SMTP (send)
	IMAPHost string
	IMAPPort int
	IMAPUser string
	IMAPPass string
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	From     string

	// Gmail API
	GmailCredentials string
	GmailToken       string
}

// New constructs a Mailbox for the given provider ("imap" | "gmail").
//
// The mailbox sub-agent replaces newIMAP/newGmail with real implementations
// (imap.go / gmail.go). Until then they return a not-implemented error so the
// rest of the tree compiles against this factory.
func New(provider string, s Settings) (Mailbox, error) {
	switch provider {
	case "imap":
		return newIMAP(s)
	case "gmail":
		return newGmail(s)
	default:
		return nil, fmt.Errorf("unknown mailbox provider %q", provider)
	}
}

func newIMAP(s Settings) (Mailbox, error)  { return newIMAPMailbox(s) }
func newGmail(s Settings) (Mailbox, error) { return newGmailMailbox(s) }
