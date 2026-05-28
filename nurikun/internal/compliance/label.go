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

	"github.com/yellowhama/musu-system/nurikun/internal/mailbox"
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

// placeholderPatterns are substrings commonly found in scaffold/template values
// that must never reach a real recipient. Hit list driven by the 2026-05-29
// incident where a self-test campaign went out with footer text
// "주소: PLACEHOLDER_실주소_미설정_운영자_수정필요" — fine for self-send, but
// would have been a clear 정보통신망법 §50 violation against any real subscriber.
//
// Matching is case-insensitive substring. Add patterns here when new template
// strings surface in init scaffolds or .env.example files.
var placeholderPatterns = []string{
	"PLACEHOLDER",
	"TODO",
	"FIXME",
	"XXX_",
	"<your-",
	"<운영자",
	"example.com",
	"example.org",
	"미설정",
	"수정필요",
}

// ValidateSenderIdentity returns an error if senderName or senderPhysical is
// empty or contains a known placeholder pattern. The campaign send path MUST
// call this before any recipient is touched: every footer renders the values
// verbatim, and a footer with "주소: PLACEHOLDER_..." both violates 정보통신망법
// §50 and burns sender reputation.
//
// The check is intentionally noisy (returns the specific bad pattern + the env
// var name) so operators see exactly what to fix.
func ValidateSenderIdentity(senderName, senderPhysical string) error {
	if strings.TrimSpace(senderName) == "" {
		return fmt.Errorf("compliance: sender_name is empty (set NURIKUN_SENDER_NAME)")
	}
	if strings.TrimSpace(senderPhysical) == "" {
		return fmt.Errorf("compliance: sender_physical is empty (set NURIKUN_SENDER_PHYSICAL); a real postal address is mandatory per 정보통신망법 §50 for the campaign footer")
	}
	for _, pat := range placeholderPatterns {
		if containsFold(senderName, pat) {
			return fmt.Errorf("compliance: sender_name still contains placeholder %q — set a real value via NURIKUN_SENDER_NAME before any send", pat)
		}
		if containsFold(senderPhysical, pat) {
			return fmt.Errorf("compliance: sender_physical still contains placeholder %q — set a real postal address via NURIKUN_SENDER_PHYSICAL before any send", pat)
		}
	}
	return nil
}

// WarnPublicBaseURL returns a non-empty warning string when publicBaseURL looks
// like a developer-only address (localhost, 127.0.0.1, etc.). Used to surface a
// "your subscribers cannot reach this URL" warning in the campaign send output.
// It does not return an error — self-sends and dry-runs are legitimate uses of
// a localhost base URL, so the operator decides whether to proceed.
func WarnPublicBaseURL(publicBaseURL string) string {
	u := strings.ToLower(strings.TrimSpace(publicBaseURL))
	for _, marker := range []string{"localhost", "127.0.0.1", "::1", "0.0.0.0"} {
		if strings.Contains(u, marker) {
			return fmt.Sprintf("public_base_url contains %q — List-Unsubscribe link will not resolve for external recipients", marker)
		}
	}
	return ""
}

// containsFold reports whether s contains substr, case-insensitively, while
// leaving non-ASCII (Korean) bytes untouched. ASCII-only fold is enough for
// the placeholder patterns above.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
