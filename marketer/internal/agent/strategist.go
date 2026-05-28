package agent

import (
	"encoding/json"
	"fmt"
)

type Strategist struct {
	Client *AgentClient
}

type MarketingBrief struct {
	ValueProp string   `json:"value_proposition"`
	Target    string   `json:"target_segment"`
	Triggers  []string `json:"triggers"`
	Goal      string   `json:"primary_goal"`
	Framework string   `json:"selected_framework"`
}

func NewStrategist(url, model, wikiDir, project string) *Strategist {
	return &Strategist{
		Client: NewAgentClient(url, model, wikiDir, project),
	}
}

func (s *Strategist) CreateBrief(context string, history string) (*MarketingBrief, error) {
	bible, err := LoadMarketingBible()
	if err != nil {
		return nil, fmt.Errorf("critical: strategy cannot proceed without marketing bible: %v", err)
	}

	var lastError error
	var currentFeedback string

	// ⚡ Agentic Self-Correction Loop (Max 3 retries)
	for attempt := 1; attempt <= 3; attempt++ {
		historySection := ""
		if history != "" {
			historySection = fmt.Sprintf("\n### PREVIOUS SUCCESSFUL CAMPAIGNS ###\n%s\n\nEnsure this new strategy builds upon the above history and does not repeat the exact same angles.", history)
		}
		
		feedbackSection := ""
		if currentFeedback != "" {
			feedbackSection = fmt.Sprintf("\n\n### PREVIOUS ERROR ###\nYour last response was invalid JSON: %s. Please fix the structure and try again.", currentFeedback)
		}

		prompt := fmt.Sprintf(`### MARKETING MASTERY BIBLE ###
%s

### MISSION ###
You are a Senior Marketing Strategist. Analyze the technical context and select the best professional strategy from the Bible.
%s
%s

Context:
%s

Tasks:
1. Select the most effective Framework (PAS, AIDA, or BAB) from the Bible.
2. Select 2 Psychological Triggers from the Bible.
3. Define the Segment and Goal.

Output in strict JSON format:
{
  "value_proposition": "...",
  "target_segment": "...",
  "triggers": ["...", "..."],
  "primary_goal": "...",
  "selected_framework": "AIDA | PAS | BAB"
}`, bible, historySection, feedbackSection, context)

		response, err := s.Client.Ask(prompt, true)
		if err != nil {
			lastError = err
			continue
		}

		var brief MarketingBrief
		if err := json.Unmarshal([]byte(response), &brief); err != nil {
			currentFeedback = err.Error()
			lastError = err
			continue
		}

		// Success!
		return &brief, nil
	}

	return nil, fmt.Errorf("strategist failed after 3 attempts: %v", lastError)
}
