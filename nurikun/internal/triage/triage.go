// Package triage classifies inbound customer-support messages into a policy
// category, detects the language, summarizes the intent, and flags sensitive
// content. The classification feeds the policy engine, which decides whether a
// drafted reply may be auto-sent or must be escalated to a human.
package triage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-nurikun/internal/agent"
	"github.com/yellowhama/musu-nurikun/internal/policy"
)

// classifyResult mirrors policy.TriageResult but with a string category so we
// can normalize/validate unknown values from the model before mapping.
type classifyResult struct {
	Category   string  `json:"category"`
	Intent     string  `json:"intent"`
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
	Sensitive  bool    `json:"sensitive"`
}

// validCategory reports whether the model returned a known category and returns
// the canonical policy.Category. Unknown values default to CategoryOther.
func normalizeCategory(raw string) policy.Category {
	switch policy.Category(strings.ToLower(strings.TrimSpace(raw))) {
	case policy.CategoryFAQ:
		return policy.CategoryFAQ
	case policy.CategoryHours:
		return policy.CategoryHours
	case policy.CategoryStatus:
		return policy.CategoryStatus
	case policy.CategorySales:
		return policy.CategorySales
	case policy.CategoryRefund:
		return policy.CategoryRefund
	case policy.CategoryComplaint:
		return policy.CategoryComplaint
	case policy.CategoryLegal:
		return policy.CategoryLegal
	default:
		return policy.CategoryOther
	}
}

// Classify asks the LLM to triage a single inbound message and returns a
// validated policy.TriageResult. Confidence is clamped to [0,1] and unknown
// categories fall back to CategoryOther so downstream policy decisions stay safe.
func Classify(client *agent.AgentClient, subject, body string) (policy.TriageResult, error) {
	prompt := buildPrompt(subject, body)

	raw, err := client.Ask(prompt, true)
	if err != nil {
		return policy.TriageResult{}, fmt.Errorf("triage classification request failed: %w", err)
	}

	var cr classifyResult
	if err := json.Unmarshal([]byte(raw), &cr); err != nil {
		return policy.TriageResult{}, fmt.Errorf("failed to parse triage JSON %q: %w", raw, err)
	}

	conf := cr.Confidence
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}

	return policy.TriageResult{
		Category:   normalizeCategory(cr.Category),
		Intent:     strings.TrimSpace(cr.Intent),
		Language:   strings.TrimSpace(cr.Language),
		Confidence: conf,
		Sensitive:  cr.Sensitive,
	}, nil
}

func buildPrompt(subject, body string) string {
	var b strings.Builder
	b.WriteString("You are a customer-support triage assistant. Classify the inbound ")
	b.WriteString("email below and respond with ONLY a JSON object (no prose, no markdown).\n\n")
	b.WriteString("Classify into exactly one of these categories:\n")
	b.WriteString("- \"faq\":       a general product/service question with a known answer\n")
	b.WriteString("- \"hours\":      asking about business hours, availability, or location\n")
	b.WriteString("- \"status\":     asking about an order/ticket/shipment status\n")
	b.WriteString("- \"sales\":      a pre-sale or purchasing inquiry\n")
	b.WriteString("- \"refund\":     a refund, billing, or money-related request\n")
	b.WriteString("- \"complaint\":  a complaint, frustration, or angry message\n")
	b.WriteString("- \"legal\":      legal threats, disputes, regulatory, or privacy demands\n")
	b.WriteString("- \"other\":      anything that does not clearly fit the above\n\n")
	b.WriteString("Also provide:\n")
	b.WriteString("- \"intent\":     a short one-sentence summary of what the customer wants\n")
	b.WriteString("- \"language\":   the ISO 639-1 code of the message language (e.g. \"en\", \"ko\")\n")
	b.WriteString("- \"confidence\": your classification confidence as a number from 0 to 1\n")
	b.WriteString("- \"sensitive\":  true for anything about money/refunds, legal matters, anger, ")
	b.WriteString("or complaints; otherwise false\n\n")
	b.WriteString("Respond with this exact shape:\n")
	b.WriteString("{\"category\":\"...\",\"intent\":\"...\",\"language\":\"...\",\"confidence\":0.0,\"sensitive\":false}\n\n")
	b.WriteString("--- EMAIL ---\n")
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n--- END EMAIL ---\n")
	return b.String()
}
