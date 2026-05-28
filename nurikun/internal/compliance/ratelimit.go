package compliance

import (
	"strings"
	"sync"
	"time"
)

// Limiter is an in-memory, per-recipient-domain send throttle. It enforces two
// independent guards that protect the sending domain's reputation:
//
//   - a minimum interval between consecutive sends to the same recipient
//     domain (so a campaign can't hammer one provider), and
//   - a per-run cap on the total number of sends allowed in this process.
//
// It is safe for concurrent use.
type Limiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	perRunCap   int
	sent        int                  // total sends allowed so far this run
	lastByDom   map[string]time.Time // last allowed send per recipient domain
	now         func() time.Time     // injectable clock (tests)
}

// NewLimiter creates a Limiter that allows at most one send per minInterval to
// any given recipient domain and at most perRunCap sends total for this run.
// A non-positive perRunCap means "no cap"; a non-positive minInterval disables
// the per-domain interval check.
func NewLimiter(minInterval time.Duration, perRunCap int) *Limiter {
	return &Limiter{
		minInterval: minInterval,
		perRunCap:   perRunCap,
		lastByDom:   make(map[string]time.Time),
		now:         time.Now,
	}
}

// domainOf extracts the lowercased domain part of an email address. If there is
// no "@", the whole (lowercased, trimmed) string is used as the bucket key.
func domainOf(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	if i := strings.LastIndex(e, "@"); i >= 0 && i < len(e)-1 {
		return e[i+1:]
	}
	return e
}

// Allow reports whether a send to email is permitted right now. When it
// returns true it records the send (consuming one unit of the per-run cap and
// resetting the domain's interval timer); when it returns false nothing is
// consumed, so the caller can simply skip this recipient.
func (l *Limiter) Allow(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Per-run cap.
	if l.perRunCap > 0 && l.sent >= l.perRunCap {
		return false
	}

	now := l.now()
	dom := domainOf(email)

	// Per-domain minimum interval.
	if l.minInterval > 0 {
		if last, ok := l.lastByDom[dom]; ok && now.Sub(last) < l.minInterval {
			return false
		}
	}

	l.lastByDom[dom] = now
	l.sent++
	return true
}

// Sent returns the number of sends allowed so far this run.
func (l *Limiter) Sent() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sent
}
