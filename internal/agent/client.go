package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ExecutionTrace struct {
	Timestamp string `json:"timestamp"`
	Project   string `json:"project"`
	Role      string `json:"role"`
	Goal      string `json:"goal"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type AgentClient struct {
	BaseURL    string
	Model      string
	WikiDir    string
	Project    string
	httpClient *http.Client
}

func NewAgentClient(baseURL, model, wikiDir, project string) *AgentClient {
	if baseURL == "" { baseURL = "http://localhost:11434/v1" }
	if model == "" { model = "llama3" }

	return &AgentClient{
		BaseURL: baseURL,
		Model:   model,
		WikiDir: wikiDir,
		Project: project,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

func (c *AgentClient) logTrace(trace ExecutionTrace) {
	date := time.Now().Format("2006-01-02")
	logDir := filepath.Join(c.WikiDir, "telemetry", date)
	os.MkdirAll(logDir, 0755)

	filename := fmt.Sprintf("%s_%s_%d.json", trace.Role, c.Project, time.Now().UnixNano())
	path := filepath.Join(logDir, filename)

	data, _ := json.MarshalIndent(trace, "", "  ")
	os.WriteFile(path, data, 0644)
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Stream         bool            `json:"stream"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func (c *AgentClient) Ask(prompt string, jsonFormat bool) (string, error) {
	trace := ExecutionTrace{
		Timestamp: time.Now().Format(time.RFC3339),
		Project:   c.Project,
		Role:      "nurikun_agent",
		Prompt:    prompt,
		Status:    "success",
	}

	reqBody := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{{Role: "user", Content: prompt}},
		Stream: false,
	}

	if jsonFormat {
		reqBody.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	jsonData, _ := json.Marshal(reqBody)
	url := c.BaseURL + "/chat/completions"

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		c.logTrace(trace)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		trace.Status = "error"
		trace.Error = string(body)
		c.logTrace(trace)
		return "", fmt.Errorf("AI error %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		c.logTrace(trace)
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		trace.Status = "error"
		trace.Error = "no choices"
		c.logTrace(trace)
		return "", fmt.Errorf("no choices")
	}

	res := chatResp.Choices[0].Message.Content
	trace.Response = res
	c.logTrace(trace)
	return res, nil
}
