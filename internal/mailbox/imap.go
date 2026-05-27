package mailbox

import (
	"fmt"
	"io"
	"log"
	"net/smtp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// imapMailbox is a Mailbox backed by IMAP (fetch/mark) and SMTP (send).
//
// It represents one real, owned address; it never spoofs or rotates
// identities. The IMAP connection is opened lazily on the first Fetch/Mark
// and reused until Close.
type imapMailbox struct {
	settings Settings
	client   *imapclient.Client
}

// newIMAPMailbox builds an IMAP/SMTP mailbox from the given settings. It does
// not open any connection yet; the IMAP client is dialed lazily so that a
// Send-only caller never needs IMAP reachability.
func newIMAPMailbox(s Settings) (*imapMailbox, error) {
	if s.IMAPHost == "" && s.SMTPHost == "" {
		return nil, fmt.Errorf("imap mailbox: no IMAP or SMTP host configured")
	}
	if s.From == "" {
		return nil, fmt.Errorf("imap mailbox: From address is required")
	}
	return &imapMailbox{settings: s}, nil
}

// connect dials and authenticates the IMAP client if it is not already open.
func (m *imapMailbox) connect() (*imapclient.Client, error) {
	if m.client != nil {
		return m.client, nil
	}
	if m.settings.IMAPHost == "" {
		return nil, fmt.Errorf("imap mailbox: IMAP host not configured")
	}

	addr := fmt.Sprintf("%s:%d", m.settings.IMAPHost, m.settings.IMAPPort)
	c, err := imapclient.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("imap mailbox: dial %s: %w", addr, err)
	}
	if err := c.Login(m.settings.IMAPUser, m.settings.IMAPPass).Wait(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("imap mailbox: login as %q: %w", m.settings.IMAPUser, err)
	}
	m.client = c
	return c, nil
}

// Fetch returns up to `limit` unseen messages from INBOX, newest first.
func (m *imapMailbox) Fetch(limit int) ([]Message, error) {
	if limit <= 0 {
		return nil, nil
	}

	c, err := m.connect()
	if err != nil {
		return nil, err
	}

	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap mailbox: select INBOX: %w", err)
	}

	// Find unprocessed (UNSEEN) messages.
	searchData, err := c.UIDSearch(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("imap mailbox: search unseen: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}

	// Keep the newest `limit` UIDs (UIDs are monotonically increasing).
	sort.Slice(uids, func(i, j int) bool { return uids[i] > uids[j] })
	if len(uids) > limit {
		uids = uids[:limit]
	}

	bodySection := &imap.FetchItemBodySection{Peek: true}
	fetchOpts := &imap.FetchOptions{
		UID:          true,
		Envelope:     true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{bodySection},
	}

	buffers, err := c.Fetch(imap.UIDSetNum(uids...), fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("imap mailbox: fetch messages: %w", err)
	}

	messages := make([]Message, 0, len(buffers))
	for _, buf := range buffers {
		msg, err := m.parseMessage(buf, bodySection)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Return newest first to match the UID ordering above.
	sort.Slice(messages, func(i, j int) bool {
		ui, _ := strconv.ParseUint(messages[i].UID, 10, 32)
		uj, _ := strconv.ParseUint(messages[j].UID, 10, 32)
		return ui > uj
	})
	return messages, nil
}

// parseMessage converts a fetched IMAP buffer into a provider-agnostic Message,
// extracting the text/plain body via go-message.
func (m *imapMailbox) parseMessage(buf *imapclient.FetchMessageBuffer, section *imap.FetchItemBodySection) (Message, error) {
	msg := Message{
		UID: strconv.FormatUint(uint64(buf.UID), 10),
	}

	if env := buf.Envelope; env != nil {
		msg.Subject = env.Subject
		msg.Date = env.Date
		msg.MessageID = env.MessageID
		if len(env.InReplyTo) > 0 {
			msg.InReplyTo = env.InReplyTo[0]
		}
		if len(env.From) > 0 {
			msg.From = env.From[0].Addr()
		}
		for i := range env.To {
			if addr := env.To[i].Addr(); addr != "" {
				msg.To = append(msg.To, addr)
			}
		}
	}
	if msg.Date.IsZero() {
		msg.Date = buf.InternalDate
	}

	raw := buf.FindBodySection(section)
	if len(raw) > 0 {
		body, hdrMsgID, hdrInReplyTo, err := extractPlainText(raw)
		if err != nil {
			return Message{}, fmt.Errorf("imap mailbox: parse body for UID %s: %w", msg.UID, err)
		}
		msg.Body = body
		// Prefer header values when the envelope did not carry them.
		if msg.MessageID == "" {
			msg.MessageID = hdrMsgID
		}
		if msg.InReplyTo == "" {
			msg.InReplyTo = hdrInReplyTo
		}
	}

	if msg.ThreadID == "" {
		msg.ThreadID = msg.UID
	}
	return msg, nil
}

// extractPlainText reads an RFC5322 message and returns its text/plain body
// along with the Message-ID and In-Reply-To header values.
func extractPlainText(raw []byte) (body, messageID, inReplyTo string, err error) {
	mr, err := mail.CreateReader(strings.NewReader(string(raw)))
	if err != nil {
		return "", "", "", fmt.Errorf("create mail reader: %w", err)
	}

	if v, e := mr.Header.Text("Message-Id"); e == nil {
		messageID = strings.Trim(v, "<>")
	}
	if v, e := mr.Header.Text("In-Reply-To"); e == nil {
		inReplyTo = strings.Trim(v, "<>")
	}

	var plain, fallback string
	for {
		p, e := mr.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", "", "", fmt.Errorf("read message part: %w", e)
		}

		inline, ok := p.Header.(*mail.InlineHeader)
		if !ok {
			continue
		}
		content, e := io.ReadAll(p.Body)
		if e != nil {
			return "", "", "", fmt.Errorf("read inline part: %w", e)
		}
		ct, _, _ := inline.ContentType()
		if ct == "text/plain" && plain == "" {
			plain = string(content)
		} else if fallback == "" {
			fallback = string(content)
		}
	}

	if plain != "" {
		return plain, messageID, inReplyTo, nil
	}
	return fallback, messageID, inReplyTo, nil
}

// Send delivers an outbound message from the owned address via SMTP.
func (m *imapMailbox) Send(out OutMessage) error {
	if m.settings.SMTPHost == "" {
		return fmt.Errorf("imap mailbox: SMTP host not configured")
	}
	if out.To == "" {
		return fmt.Errorf("imap mailbox: outbound message has no recipient")
	}

	raw := buildRFC5322(m.settings.From, out)

	addr := fmt.Sprintf("%s:%d", m.settings.SMTPHost, m.settings.SMTPPort)
	var auth smtp.Auth
	if m.settings.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.settings.SMTPUser, m.settings.SMTPPass, m.settings.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, m.settings.From, []string{out.To}, raw); err != nil {
		return fmt.Errorf("imap mailbox: send mail via %s: %w", addr, err)
	}
	return nil
}

// buildRFC5322 assembles a plain-text RFC5322 message with CRLF line endings.
func buildRFC5322(from string, out OutMessage) []byte {
	var b strings.Builder

	writeHeader := func(name, value string) {
		if value == "" {
			return
		}
		// Strip any embedded newlines to prevent header injection.
		value = strings.NewReplacer("\r", "", "\n", "").Replace(value)
		fmt.Fprintf(&b, "%s: %s\r\n", name, value)
	}

	writeHeader("From", from)
	writeHeader("To", out.To)
	writeHeader("Subject", out.Subject)
	writeHeader("Date", time.Now().Format(time.RFC1123Z))
	if out.InReplyTo != "" {
		id := out.InReplyTo
		if !strings.HasPrefix(id, "<") {
			id = "<" + id + ">"
		}
		writeHeader("In-Reply-To", id)
		writeHeader("References", id)
	}
	for k, v := range out.Headers {
		// Do not let caller-supplied headers override the core ones above.
		switch strings.ToLower(k) {
		case "from", "to", "subject", "date", "mime-version", "content-type":
			continue
		}
		writeHeader(k, v)
	}
	writeHeader("MIME-Version", "1.0")
	writeHeader("Content-Type", "text/plain; charset=UTF-8")

	b.WriteString("\r\n")
	// Normalize body line endings to CRLF.
	body := strings.ReplaceAll(out.Body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")
	b.WriteString(body)

	return []byte(b.String())
}

// Mark flags the given UID as \Seen so it is not fetched again.
func (m *imapMailbox) Mark(uid string) error {
	n, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		return fmt.Errorf("imap mailbox: invalid UID %q: %w", uid, err)
	}

	c, err := m.connect()
	if err != nil {
		return err
	}
	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		return fmt.Errorf("imap mailbox: select INBOX: %w", err)
	}

	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagSeen},
		Silent: true,
	}
	if err := c.Store(imap.UIDSetNum(imap.UID(uint32(n))), storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("imap mailbox: mark UID %s seen: %w", uid, err)
	}
	return nil
}

// Close logs out and releases the IMAP connection.
func (m *imapMailbox) Close() error {
	if m.client == nil {
		return nil
	}
	c := m.client
	m.client = nil

	if err := c.Logout().Wait(); err != nil {
		// Logout can fail if the connection is already gone; still close.
		log.Printf("imap mailbox: logout: %v", err)
		_ = c.Close()
		return fmt.Errorf("imap mailbox: logout: %w", err)
	}
	if err := c.Close(); err != nil {
		return fmt.Errorf("imap mailbox: close: %w", err)
	}
	return nil
}
