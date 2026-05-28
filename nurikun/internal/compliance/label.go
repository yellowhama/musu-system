// Package compliance enforces the legal guardrails for outbound opt-in
// campaigns: commercial-advertisement labeling, a sender-identity + postal
// footer, one-click unsubscribe headers, and a per-recipient rate limiter.
//
// These are not optional decorations. Under Korea's 정보통신망법 §50
// (Act on Promotion of Information and Communications Network Utilization),
// commercial advertisement email must carry an "(광고)" subject prefix, the
// sender's identity and physical address, and an easy opt-out mechanism. The
// campaign sender applies all of these unconditionally, which is also what
// keeps the sending domain deliverable (List-Unsubscribe is honored by major
// inbox providers).
package compliance

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/yellowhama/musu-nurikun/internal/mailbox"
)

// adPrefix is the mandatory commercial-advertisement label (정보통신망법 §50).
const adPrefix = "(광고) "

// EnsureAdPrefix prepends the commercial-advertisement label to subject if it
// is not already present. It is idempotent: a subject that already carries the
// "(광고)" marker (with or without the trailing space) is returned unchanged.
func EnsureAdPrefix(subject string) string {
	trimmed := strings.TrimSpace(subject)
	// Already labeled (tolerate "(광고)" with or without a following space).
	if strings.HasPrefix(trimmed, "(광고)") {
		return subject
	}
	return adPrefix + subject
}

// Footer returns the plain-text compliance footer appended to every outbound
// campaign body. It identifies the sender, states the postal address, and
// gives a one-click unsubscribe line so recipients can always opt out.
func Footer(senderName, senderPostalAddress, unsubscribeURL string) string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString("--\n")
	b.WriteString("이 메일은 수신에 동의(opt-in)하신 분께만 발송됩니다.\n")
	if senderName != "" {
		b.WriteString(fmt.Sprintf("보낸 사람: %s\n", senderName))
	}
	if senderPostalAddress != "" {
		b.WriteString(fmt.Sprintf("주소: %s\n", senderPostalAddress))
	}
	if unsubscribeURL != "" {
		b.WriteString(fmt.Sprintf("수신거부(unsubscribe): %s\n", unsubscribeURL))
	}
	return b.String()
}

// UnsubscribeHeader builds the value for the RFC 8058 / RFC 2369
// List-Unsubscribe header. URLs and mailto: targets are wrapped in angle
// brackets and comma-separated. A bare email address is promoted to a
// mailto: URI; anything else is treated as a URL.
func UnsubscribeHeader(unsubscribeURL string) string {
	target := strings.TrimSpace(unsubscribeURL)
	if target == "" {
		return ""
	}

	var parts []string
	if strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		parts = append(parts, fmt.Sprintf("<%s>", target))
	} else if _, err := mail.ParseAddress(target); err == nil {
		// A bare address — expose it as a one-click mailto unsubscribe.
		parts = append(parts, fmt.Sprintf("<mailto:%s>", target))
	} else {
		// Fall back to treating it as a URL.
		parts = append(parts, fmt.Sprintf("<%s>", target))
	}
	return strings.Join(parts, ", ")
}

// Decorate applies all mandatory compliance transforms to an outbound message
// in place: the "(광고)" subject prefix, the sender/postal/unsubscribe footer,
// and the List-Unsubscribe header. Callers must invoke this for every campaign
// message; there is intentionally no flag to skip it.
func Decorate(out *mailbox.OutMessage, senderName, postal, unsubURL string) {
	if out == nil {
		return
	}
	out.Subject = EnsureAdPrefix(out.Subject)
	out.Body += Footer(senderName, postal, unsubURL)

	if h := UnsubscribeHeader(unsubURL); h != "" {
		if out.Headers == nil {
			out.Headers = make(map[string]string)
		}
		out.Headers["List-Unsubscribe"] = h
		// RFC 8058 one-click signal; harmless when the unsubscribe target is a URL.
		out.Headers["List-Unsubscribe-Post"] = "List-Unsubscribe=One-Click"
	}
}
