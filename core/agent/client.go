// Package agent is the shared OpenAI-compatible LLM client used by every CLI
// in the Musu ecosystem. It unifies the previously-triplicated AgentClient
// implementations (crawl-ai, marketer, nurikun) behind a single type with
// option-shaped variations:
//
//   - WithVisionModel  enables DescribeImage against a separate vision model
//                      (used by crawl-ai's image-aware harvesters)
//   - WithTelemetry    enables per-call JSON trace persistence under
//                      <WikiDir>/telemetry/<date>/ (used by all three when a
//                      WikiDir is available)
//   - WithRole         overrides the default "assistant" role label written
//                      to traces (nurikun tags its calls "nurikun_agent")
//   - WithHTTPClient   injects a custom *http.Client (for tests)
//
// Telemetry never aborts the main flow: mkdir/write failures are reported to
// stderr and the chat request still returns its result.
package agent

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ExecutionTrace is one chat round's audit record. Serialized to JSON when
// telemetry is enabled.
type ExecutionTrace struct {
	Timestamp string `json:"timestamp"`
	Project   string `json:"project"`
	Role      string `json:"role"`
	Goal      string `json:"goal"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Status    string `json:"status"` // success | error
	Error     string `json:"error,omitempty"`
}

// Client is the unified OpenAI-compatible chat/embed/vision client.
type Client struct {
	BaseURL     string
	Model       string
	VisionModel string
	WikiDir     string // empty disables telemetry
	Project     string
	Role        string // role label written to ExecutionTrace
	httpClient  *http.Client
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithVisionModel sets the model used by DescribeImage. Defaults to "llava".
func WithVisionModel(model string) Option {
	return func(c *Client) {
		if model != "" {
			c.VisionModel = model
		}
	}
}

// WithTelemetry enables per-call JSON trace persistence under
// <wikiDir>/telemetry/<YYYY-MM-DD>/. Passing an empty wikiDir is a no-op
// (telemetry stays disabled).
func WithTelemetry(wikiDir, project string) Option {
	return func(c *Client) {
		c.WikiDir = wikiDir
		c.Project = project
	}
}

// WithRole overrides the default "assistant" role label.
func WithRole(role string) Option {
	return func(c *Client) {
		if role != "" {
			c.Role = role
		}
	}
}

// WithHTTPClient injects a custom *http.Client. Intended for tests.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// New constructs a Client. baseURL "" → http://localhost:11434/v1;
// model "" → llama3. Apply options for telemetry/vision/role/httpClient.
func New(baseURL, model string, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	if model == "" {
		model = "llama3"
	}
	c := &Client{
		BaseURL:     baseURL,
		Model:       model,
		VisionModel: "llava",
		Role:        "assistant",
		httpClient: &http.Client{
			Timeout: clientTimeout(),
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 20,
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// clientTimeout is the HTTP timeout for one chat round. Local LLM long-form
// generation (and self-correction rewrite loops) routinely exceeds the old
// 120s; the default is 300s, overridable via MUSU_AGENT_TIMEOUT_SECONDS for
// slower hardware or larger models.
func clientTimeout() time.Duration {
	if v := os.Getenv("MUSU_AGENT_TIMEOUT_SECONDS"); v != "" {
		if n, err := time.ParseDuration(v + "s"); err == nil && n > 0 {
			return n
		}
	}
	return 300 * time.Second
}

// telemetryEnabled reports whether trace writes should be attempted.
func (c *Client) telemetryEnabled() bool { return c.WikiDir != "" }

func (c *Client) logTrace(trace ExecutionTrace) {
	if !c.telemetryEnabled() {
		return
	}
	date := time.Now().Format("2006-01-02")
	logDir := filepath.Join(c.WikiDir, "telemetry", date)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "telemetry: mkdir %s: %v\n", logDir, err)
		return
	}
	filename := fmt.Sprintf("%s_%s_%d.json", trace.Role, c.Project, time.Now().UnixNano())
	path := filepath.Join(logDir, filename)
	data, _ := json.MarshalIndent(trace, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "telemetry: write %s: %v\n", path, err)
	}
}

// --- OpenAI wire types ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Stream         bool            `json:"stream"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Ask sends a chat completion request and (if telemetry enabled) records a trace.
func (c *Client) Ask(prompt string, jsonFormat bool) (string, error) {
	trace := ExecutionTrace{
		Timestamp: time.Now().Format(time.RFC3339),
		Project:   c.Project,
		Role:      c.Role,
		Prompt:    prompt,
		Status:    "success",
	}

	reqBody := chatRequest{
		Model:    c.Model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	if jsonFormat {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := c.httpClient.Post(c.BaseURL+"/chat/completions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		c.logTrace(trace)
		return "", fmt.Errorf("AI connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		trace.Status = "error"
		trace.Error = string(body)
		c.logTrace(trace)
		return "", fmt.Errorf("AI returned error %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		trace.Status = "error"
		trace.Error = err.Error()
		c.logTrace(trace)
		return "", fmt.Errorf("failed to decode AI response: %v", err)
	}
	if len(out.Choices) == 0 {
		trace.Status = "error"
		trace.Error = "no choices returned"
		c.logTrace(trace)
		return "", fmt.Errorf("AI returned no choices")
	}

	res := out.Choices[0].Message.Content
	trace.Response = res
	c.logTrace(trace)
	return res, nil
}

// --- Embeddings ---

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// Embed generates an embedding vector for the given text. Not telemetry-traced
// because callers typically embed in bulk and tracing every vector is noise.
func (c *Client) Embed(text string) ([]float64, error) {
	reqBody := embedRequest{Model: c.Model, Input: text}
	jsonData, _ := json.Marshal(reqBody)
	resp, err := c.httpClient.Post(c.BaseURL+"/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("AI embed failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI embed error %d: %s", resp.StatusCode, string(body))
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode AI embed response: %v", err)
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("AI returned no embeddings")
	}
	return out.Data[0].Embedding, nil
}

// --- Vision ---

type visionImageURL struct {
	URL string `json:"url"`
}

type visionContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *visionImageURL `json:"image_url,omitempty"`
}

type visionMessage struct {
	Role    string          `json:"role"`
	Content []visionContent `json:"content"`
}

type visionRequest struct {
	Model    string          `json:"model"`
	Messages []visionMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// DescribeImage returns a one-sentence description of the image file using the
// configured VisionModel. Used by crawl-ai's image-aware harvesters.
func (c *Client) DescribeImage(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	imageURL := fmt.Sprintf("data:image/png;base64,%s", b64)

	reqBody := visionRequest{
		Model: c.VisionModel,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionContent{
					{Type: "text", Text: "Describe this image in one concise sentence for a knowledge base index."},
					{Type: "image_url", ImageURL: &visionImageURL{URL: imageURL}},
				},
			},
		},
		Stream: false,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := c.httpClient.Post(c.BaseURL+"/chat/completions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("AI vision failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI vision error %d: %s", resp.StatusCode, string(body))
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode AI vision response: %v", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("AI vision returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}
