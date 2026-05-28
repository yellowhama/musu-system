package agent

import (
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/nurikun/internal/knowledge"
)

// Respond composes a grounded customer-support reply. The reply is anchored in
// the provided knowledge snippets: the model must not invent facts beyond them.
// When no snippets are available it answers conservatively and suggests that a
// human follow up. The reply is written in the customer's language and, when a
// persona/tone is given, in that voice.
//
// It returns the reply text and errors if the model produces an empty reply.
func Respond(client *AgentClient, customerMsg string, snippets []knowledge.Snippet, persona string) (string, error) {
	prompt := buildResponsePrompt(customerMsg, snippets, persona)

	reply, err := client.Ask(prompt, false)
	if err != nil {
		return "", fmt.Errorf("reply generation request failed: %w", err)
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("AI returned an empty reply")
	}
	return reply, nil
}

func buildResponsePrompt(customerMsg string, snippets []knowledge.Snippet, persona string) string {
	var b strings.Builder
	b.WriteString("You are a first-party customer-support agent replying to a customer's email.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Write the reply in the SAME language the customer used.\n")
	b.WriteString("- Ground every factual claim in the KNOWLEDGE section below. ")
	b.WriteString("Do NOT invent facts, prices, policies, dates, or commitments that are not present there.\n")
	b.WriteString("- If the knowledge is insufficient to fully answer, say so honestly, ")
	b.WriteString("answer only what you safely can, and offer to have a human colleague follow up.\n")
	b.WriteString("- Be concise, warm, and helpful. Do not fabricate citations or links.\n")

	if strings.TrimSpace(persona) != "" {
		b.WriteString("- Write in this persona/tone: ")
		b.WriteString(strings.TrimSpace(persona))
		b.WriteString("\n")
	}

	b.WriteString("- Output only the reply body text. Do not include a subject line or email headers.\n\n")

	b.WriteString("--- KNOWLEDGE ---\n")
	if len(snippets) == 0 {
		b.WriteString("(no grounding available — answer conservatively and suggest a human follow-up)\n")
	} else {
		for i, s := range snippets {
			fmt.Fprintf(&b, "[%d] %s", i+1, strings.TrimSpace(s.Title))
			if s.Source != "" {
				fmt.Fprintf(&b, " (source: %s)", s.Source)
			}
			b.WriteString("\n")
			b.WriteString(strings.TrimSpace(s.Content))
			b.WriteString("\n\n")
		}
	}
	b.WriteString("--- END KNOWLEDGE ---\n\n")

	b.WriteString("--- CUSTOMER MESSAGE ---\n")
	b.WriteString(strings.TrimSpace(customerMsg))
	b.WriteString("\n--- END CUSTOMER MESSAGE ---\n\n")
	b.WriteString("Write the reply now:")
	return b.String()
}
