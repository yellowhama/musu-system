package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SoulForger struct {
	OllamaURL  string
	Model      string
	httpClient *http.Client
}

type SoulProfile struct {
	Job       string   `json:"job"`
	Backstory string   `json:"backstory"`
	Tone      string   `json:"tone"`
	Interests []string `json:"interests"`
	BioShort  string   `json:"bio_short"`
}

func NewSoulForger(model string) *SoulForger {
	return &SoulForger{
		OllamaURL: "http://localhost:11434/api/generate",
		Model:     model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (s *SoulForger) GenerateSoul(name string) (*SoulProfile, error) {
	prompt := fmt.Sprintf(`You are a Character Designer. Generate a realistic, deep internet persona for a person named "%s".
This person should feel like a real human, not a generic bot.

Output in strict JSON format:
{
  "job": "specific profession",
  "backstory": "2-3 sentences about their life and how they ended up in their current field",
  "tone": "describe their writing style (e.g., cynical, highly enthusiastic, professional yet witty)",
  "interests": ["interest 1", "interest 2", "interest 3"],
  "bio_short": "A 160-character bio suitable for X or Reddit"
}`, name)

	reqBody := map[string]interface{}{
		"model":  s.Model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := s.httpClient.Post(s.OllamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }

	var profile SoulProfile
	if err := json.Unmarshal([]byte(result.Response), &profile); err != nil {
		return nil, fmt.Errorf("failed to parse soul JSON: %v", err)
	}

	return &profile, nil
}
