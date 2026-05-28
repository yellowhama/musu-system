package agent

import (
	"encoding/json"
	"fmt"
)

type Critic struct {
	Client *AgentClient
}

type CriticEvaluation struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback"`
}

func NewCritic(url, model, wikiDir, project string) *Critic {
	return &Critic{
		Client: NewAgentClient(url, model, wikiDir, project),
	}
}

func (c *Critic) Evaluate(brief *MarketingBrief, personaContent, draft string) (*CriticEvaluation, error) {
	briefJSON, _ := json.MarshalIndent(brief, "", "  ")
	
	prompt := fmt.Sprintf(`### MARKETING MASTERY AUDITOR ###
You are a Zero-Tolerance Marketing Editor and AI Detection Specialist.

### REFERENCE DOCUMENTS ###
STRATEGIC BRIEF:
%s

PERSONA PROFILE:
%s

### CONTENT TO AUDIT ###
%s

### AUDIT CRITERIA ###
1. NO AI FLUFF: Reject any generic filler (e.g., "In today's digital world", "Unlock the power").
2. STRATEGY ALIGNMENT: Does it follow the Framework (%s) and Triggers (%v)?
3. HOOK STRENGTH: Is the first sentence a world-class viral hook?
4. PERSONA INTEGRITY: Does the voice match the profile perfectly?

### OUTPUT INSTRUCTIONS ###
Output in strict JSON format:
{
  "approved": true | false,
  "feedback": "Detailed explanation of flaws or suggestions for improvement"
}`, string(briefJSON), personaContent, draft, brief.Framework, brief.Triggers)

	response, err := c.Client.Ask(prompt, true)
	if err != nil { return nil, err }

	return parseCriticEvaluation(response)
}

// parseCriticEvaluation decodes the raw model response into a CriticEvaluation.
func parseCriticEvaluation(response string) (*CriticEvaluation, error) {
	var eval CriticEvaluation
	if err := json.Unmarshal([]byte(response), &eval); err != nil {
		return nil, fmt.Errorf("failed to parse critic evaluation JSON: %v", err)
	}
	return &eval, nil
}
