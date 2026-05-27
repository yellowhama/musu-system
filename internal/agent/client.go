package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AgentClient is a generic OpenAI-compatible LLM client. It targets an `ai_url`
// such as Ollama's http://localhost:11434/v1 or any OpenAI-compatible server.
type AgentClient struct {
	BaseURL    string
	Model      string
	httpClient *http.Client
}

func NewAgentClient(baseURL, model string) *AgentClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	if model == "" {
		model = "llama3"
	}
	return &AgentClient{
		BaseURL: baseURL,
		Model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 20,
			},
		},
	}
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Stream         bool            `json:"stream"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// Ask sends a single user prompt. When jsonFormat is true it requests a strict
// JSON object response.
func (c *AgentClient) Ask(prompt string, jsonFormat bool) (string, error) {
	reqBody := chatRequest{
		Model:    c.Model,
		Messages: []ChatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	if jsonFormat {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := c.httpClient.Post(c.BaseURL+"/chat/completions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("AI connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI returned error %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode AI response: %v", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("AI returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
