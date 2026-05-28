package policy

import "fmt"

// Category is the classification bucket for an inbound message.
type Category string

const (
	CategoryFAQ       Category = "faq"
	CategoryHours     Category = "hours"
	CategoryStatus    Category = "status"
	CategorySales     Category = "sales"
	CategoryRefund    Category = "refund"
	CategoryComplaint Category = "complaint"
	CategoryLegal     Category = "legal"
	CategoryOther     Category = "other"
)

// TriageResult is the classification of an inbound message, produced by the
// triage package and consumed by the policy engine. Kept here (the consumer's
// package) so triage -> policy is a one-way dependency with no import cycle.
type TriageResult struct {
	Category   Category `json:"category"`
	Intent     string   `json:"intent"`
	Language   string   `json:"language"`
	Confidence float64  `json:"confidence"`
	Sensitive  bool     `json:"sensitive"`
}

// Decision is the outcome of applying policy to a triage result.
type Decision struct {
	AutoSend bool
	Escalate bool
	Reason   string
}

// Config controls when a drafted reply may be sent without human review.
type Config struct {
	ConfidenceThreshold float64    `json:"confidence_threshold"`
	AutoAllowlist       []Category `json:"auto_allowlist"`
	EscalateCategories  []Category `json:"escalate_categories"`
}

// DefaultConfig is a safe starting point: only low-risk informational categories
// auto-send, and money/legal/complaints always escalate to a human.
func DefaultConfig() Config {
	return Config{
		ConfidenceThreshold: 0.85,
		AutoAllowlist:       []Category{CategoryFAQ, CategoryHours, CategoryStatus},
		EscalateCategories:  []Category{CategoryRefund, CategoryComplaint, CategoryLegal},
	}
}

// Decide returns whether a reply may be auto-sent, must be escalated, or should
// be drafted for review. Escalation always wins over auto-send.
func (c Config) Decide(t TriageResult) Decision {
	for _, cat := range c.EscalateCategories {
		if t.Category == cat {
			return Decision{Escalate: true, Reason: fmt.Sprintf("category %q requires human review", cat)}
		}
	}
	if t.Sensitive {
		return Decision{Escalate: true, Reason: "message flagged sensitive"}
	}
	if t.Confidence >= c.ConfidenceThreshold {
		for _, cat := range c.AutoAllowlist {
			if t.Category == cat {
				return Decision{AutoSend: true, Reason: fmt.Sprintf("confidence %.2f >= %.2f and %q allow-listed", t.Confidence, c.ConfidenceThreshold, cat)}
			}
		}
	}
	return Decision{Reason: fmt.Sprintf("confidence %.2f / category %q not auto-eligible - draft for review", t.Confidence, t.Category)}
}
