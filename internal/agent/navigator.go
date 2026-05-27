package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/playwright-community/playwright-go"
)

type Navigator struct {
	OllamaURL  string
	Model      string
	httpClient *http.Client // Optimized: Reusable client pool
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
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (n *Navigator) DetermineNextAction(goal string, page playwright.Page) (*BrowserAction, error) {
	// 1. Extract high-resolution DOM (including roles and aria labels)
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

	domData, _ := json.Marshal(domJSON)

	// 2. Prompt LLM with enhanced context
	prompt := fmt.Sprintf(`You are an Expert Browser Navigator.
Goal: %s
Current URL: %s

Interactive Elements (JSON):
%s

Identify the next step to reach the goal. Use specific selectors like [id='...'], [name='...'], or text='...'.
Output in strict JSON:
{
  "action": "click | fill | wait | done",
  "selector": "CSS selector",
  "value": "text (if filling)",
  "reason": "short explanation"
}`, goal, page.URL(), string(domData))

	reqBody := map[string]interface{}{
		"model":  n.Model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := n.httpClient.Post(n.OllamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error: %s", string(body))
	}

	var result struct {
		Response string `json:"response"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	var action BrowserAction
	if err := json.Unmarshal([]byte(result.Response), &action); err != nil {
		return nil, fmt.Errorf("failed to parse navigator JSON: %v", err)
	}

	return &action, nil
}
