package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Navigator struct {
	OllamaURL  string
	Model      string
	httpClient *http.Client
}

type BrowserAction struct {
	Action   string `json:"action"` // click, fill, wait, done
	Selector string `json:"selector"`
	Value    string `json:"value"`
	Reason   string `json:"reason"`
}

func NewNavigator(model string) *Navigator {
	return &Navigator{
		OllamaURL: "http://localhost:11434/api/generate",
		Model:     model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

func (n *Navigator) DetermineNextAction(goal string, page playwright.Page) (*BrowserAction, error) {
	// 1. Extract high-resolution DOM
	domJSON, err := page.Evaluate(`() => {
		const elements = document.querySelectorAll('button, input, a, select, [role="button"], [role="link"]');
		return Array.from(elements).map(el => {
			const box = el.getBoundingClientRect();
			return {
				tag: el.tagName,
				id: el.id,
				name: el.getAttribute('name'),
				type: el.getAttribute('type'),
				role: el.getAttribute('role'),
				aria: el.getAttribute('aria-label'),
				text: (el.innerText || el.value || "").slice(0, 50),
				visible: box.width > 0 && box.height > 0
			};
		}).filter(e => e.visible).slice(0, 60);
	}`)
	if err != nil { return nil, err }

	domData, _ := json.MarshalIndent(domJSON, "", "  ")

	var lastError error
	var currentFeedback string

	// ⚡ Agentic Self-Correction & Handover Loop
	for attempt := 1; attempt <= 3; attempt++ {
		feedbackSection := ""
		if currentFeedback != "" {
			feedbackSection = fmt.Sprintf("\n\n### PREVIOUS ERROR ###\nYour last response was invalid: %s.", currentFeedback)
		}

		prompt := fmt.Sprintf(`### MISSION ###
Goal: %s
Current URL: %s

### DOM ELEMENTS ###
%s
%s

Identify the next action. Output in strict JSON.`, goal, page.URL(), string(domData), feedbackSection)

		reqBody := map[string]interface{}{
			"model":  n.Model,
			"prompt": prompt,
			"stream": false,
			"format": "json",
		}

		jsonData, _ := json.Marshal(reqBody)
		
		// 🛠️ SURGICAL FIX: Do not return 'err' directly. Wrap it in AGENT_REQUIRED to trigger handover.
		resp, err := n.httpClient.Post(n.OllamaURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("AGENT_REQUIRED: Ollama unreachable: %v. DOM:\n%s", err, string(domData))
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("AGENT_REQUIRED: Ollama error %d. DOM:\n%s", resp.StatusCode, string(domData))
		}

		var result struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			currentFeedback = "JSON Decode failed"
			lastError = err
			continue
		}

		var action BrowserAction
		if err := json.Unmarshal([]byte(result.Response), &action); err != nil {
			currentFeedback = "Invalid JSON structure"
			lastError = err
			continue
		}

		return &action, nil
	}

	return nil, fmt.Errorf("AGENT_REQUIRED: All attempts failed (%v). DOM:\n%s", lastError, string(domData))
}
