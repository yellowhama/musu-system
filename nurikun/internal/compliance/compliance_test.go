package compliance

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yellowhama/musu-nurikun/internal/mailbox"
)

func TestEnsureAdPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain subject gets prefix", "Spring sale", "(광고) Spring sale"},
		{"already prefixed unchanged", "(광고) Spring sale", "(광고) Spring sale"},
		{"prefixed without space unchanged", "(광고)Spring sale", "(광고)Spring sale"},
		{"empty subject still labeled", "", "(광고) "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnsureAdPrefix(tt.in); got != tt.want {
				t.Errorf("EnsureAdPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEnsureAdPrefixIdempotent(t *testing.T) {
	once := EnsureAdPrefix("Newsletter")
	twice := EnsureAdPrefix(once)
	if once != twice {
		t.Errorf("EnsureAdPrefix not idempotent: once=%q twice=%q", once, twice)
	}
	if strings.Count(twice, "(광고)") != 1 {
		t.Errorf("double prefix detected: %q", twice)
	}
}

func TestFooterContainsIdentity(t *testing.T) {
	f := Footer("Acme Co", "123 Main St, Seoul", "https://example.com/u/abc")
	for _, want := range []string{"Acme Co", "123 Main St, Seoul", "https://example.com/u/abc"} {
		if !strings.Contains(f, want) {
			t.Errorf("footer missing %q:\n%s", want, f)
		}
	}
}

func TestUnsubscribeHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"url", "https://example.com/u/abc", "<https://example.com/u/abc>"},
		{"bare email becomes mailto", "unsub@example.com", "<mailto:unsub@example.com>"},
		{"mailto preserved", "mailto:unsub@example.com", "<mailto:unsub@example.com>"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnsubscribeHeader(tt.in); got != tt.want {
				t.Errorf("UnsubscribeHeader(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecorate(t *testing.T) {
	out := &mailbox.OutMessage{
		To:      "reader@gmail.com",
		Subject: "Weekly update",
		Body:    "Hello!",
	}
	Decorate(out, "Acme Co", "123 Main St", "https://example.com/u/abc")

	if !strings.HasPrefix(out.Subject, "(광고) ") {
		t.Errorf("subject not labeled: %q", out.Subject)
	}
	if !strings.Contains(out.Body, "https://example.com/u/abc") {
		t.Errorf("body missing unsubscribe URL: %q", out.Body)
	}
	if !strings.HasPrefix(out.Body, "Hello!") {
		t.Errorf("original body not preserved: %q", out.Body)
	}
	if got := out.Headers["List-Unsubscribe"]; got != "<https://example.com/u/abc>" {
		t.Errorf("List-Unsubscribe = %q", got)
	}

	// Idempotent on subject: decorating again must not double-prefix.
	Decorate(out, "Acme Co", "123 Main St", "https://example.com/u/abc")
	if strings.Count(out.Subject, "(광고)") != 1 {
		t.Errorf("subject double-prefixed after second Decorate: %q", out.Subject)
	}
}

func TestLimiterPerRunCap(t *testing.T) {
	l := NewLimiter(0, 2) // no interval, cap of 2

	if !l.Allow("a@x.com") {
		t.Fatal("1st send should be allowed")
	}
	if !l.Allow("b@y.com") {
		t.Fatal("2nd send should be allowed")
	}
	if l.Allow("c@z.com") {
		t.Error("3rd send should be blocked by per-run cap")
	}
	if l.Sent() != 2 {
		t.Errorf("Sent() = %d, want 2", l.Sent())
	}
}

func TestLimiterPerDomainInterval(t *testing.T) {
	l := NewLimiter(time.Hour, 0) // 1h interval, no cap
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cur := base
	l.now = func() time.Time { return cur }

	if !l.Allow("first@example.com") {
		t.Fatal("first send to domain should be allowed")
	}
	// Same domain, too soon.
	if l.Allow("second@example.com") {
		t.Error("second send to same domain within interval should be blocked")
	}
	// Different domain is unaffected.
	if !l.Allow("other@elsewhere.com") {
		t.Error("send to different domain should be allowed")
	}
	// Advance past the interval; same domain allowed again.
	cur = base.Add(time.Hour + time.Minute)
	if !l.Allow("third@example.com") {
		t.Error("send after interval elapsed should be allowed")
	}
}

func TestLimiterConcurrent(t *testing.T) {
	l := NewLimiter(0, 100)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Allow("x@concurrent.com")
		}()
	}
	wg.Wait()
	if l.Sent() != 100 {
		t.Errorf("Sent() = %d, want 100 (cap)", l.Sent())
	}
}
