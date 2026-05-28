package mailbox

import (
	"fmt"
	"strings"
	"sync"
)

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
type Factory func(Settings) (Mailbox, error)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]Factory{
		"imap":  newIMAP,
		"gmail": newGmail,
	}
)

func New(provider string, s Settings) (Mailbox, error) {
	name := strings.ToLower(strings.TrimSpace(provider))

	factoriesMu.RLock()
	factory, ok := factories[name]
	factoriesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown mailbox provider %q", provider)
	}
	return factory(s)
}

// RegisterFactory installs or overrides a provider factory. It returns a restore
// function so tests can cleanly roll back temporary registrations.
func RegisterFactory(name string, factory Factory) func() {
	key := strings.ToLower(strings.TrimSpace(name))
	factoriesMu.Lock()
	previous, hadPrevious := factories[key]
	factories[key] = factory
	factoriesMu.Unlock()

	return func() {
		factoriesMu.Lock()
		defer factoriesMu.Unlock()
		if hadPrevious {
			factories[key] = previous
			return
		}
		delete(factories, key)
	}
}

func newIMAP(s Settings) (Mailbox, error)  { return newIMAPMailbox(s) }
func newGmail(s Settings) (Mailbox, error) { return newGmailMailbox(s) }
