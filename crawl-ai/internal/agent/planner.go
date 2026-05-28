package agent

import (
	"encoding/json"
	"fmt"
)

type Planner struct {
	Client        *AgentClient
	CustomPersona string
}

type SearchPlan struct {
	Queries    []string `json:"queries"`
	Hypotheses []string `json:"hypotheses"`
	Reason     string   `json:"reason"`
}

func (p *Planner) CreatePlan(userQuery string) (*SearchPlan, error) {
	prompt := fmt.Sprintf(`You are a Socratic Research Planner for "musu-crawl-ai". 
Your goal is not just to find information, but to rigorously test truths.

1. DECONSTRUCT: Break the user question into core assumptions and hypotheses.
2. FALSIFY: Design search queries specifically to find evidence that might REFUTE those assumptions, as well as support them.
3. DIVERSIFY: Ensure queries target Arxiv for theory, GitHub for implementation, and Reddit/X for real-world sentiment.

%s

User Question: %s

Output your response in strict JSON format:
{
  "queries": ["falsification query 1", "supporting query 2", ...],
  "hypotheses": ["hidden assumption 1", "testable hypothesis 2", ...],
  "reason": "explanation of the critical thinking strategy"
}`, p.CustomPersona, userQuery)

	response, err := p.Client.Ask(prompt, true)
	if err != nil {
		return nil, err
	}

	var plan SearchPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse planner JSON: %v (response was: %s)", err, response)
	}

	return &plan, nil
}
