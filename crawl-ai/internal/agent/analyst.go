package agent

import (
	"fmt"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type Analyst struct {
	Client        *AgentClient
	CustomPersona string
}

type SourceContent struct {
	ID          string
	Content     string
	Reliability float64
	SourceType  string
}

func (a *Analyst) Synthesize(query string, sources []SourceContent) (string, error) {
	fullContext := ""
	for i, s := range sources {
		content := s.Content
		// If single source is huge, summarize it first locally
		if len(content) > 4000 {
			content = utils.Summarize(content, 10) // Get top 10 sentences
		}

		fullContext += fmt.Sprintf("--- Source %d [%s] (Reliability: %.1f) ---\n%s\n\n", i+1, s.SourceType, s.Reliability, content)
		if len(fullContext) > 15000 { // Increased limit for complex research
			break
		}
	}

	prompt := fmt.Sprintf(`You are a Skeptical Research Analyst for "musu-crawl-ai". 

%s

Below are several sources collected for the query: "%s"
Note the reliability scores (0.0 to 1.0). Higher is better (e.g. Arxiv=0.9, Reddit=0.5).

Sources:
%s

Your task:
1. CROSS-VERIFY: Look for contradictions between sources. If Source A says X and Source B says Y, highlight this.
2. SYNTHESIZE: Provide a comprehensive answer, weighting high-reliability sources more heavily.
3. IDENTIFY GAPS: State what is still missing or remains uncertain.

Format your output as:
ANSWER: [Your synthesized response]
CONTRADICTIONS: [Description of any conflicting info, or "None"]
MISSING: [List of specific information gaps, or "None"]`, a.CustomPersona, query, fullContext)

	return a.Client.Ask(prompt, false)
}
