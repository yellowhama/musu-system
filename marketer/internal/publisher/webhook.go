package publisher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookPublisher struct {
	TargetURL string
}

func NewWebhookPublisher(url string) *WebhookPublisher {
	return &WebhookPublisher{TargetURL: url}
}

func (p *WebhookPublisher) Publish(topic, content string) (string, error) {
	if p.TargetURL == "" {
		return "", fmt.Errorf("webhook target URL not set")
	}

	payload := map[string]string{
		"topic":     topic,
		"content":   content,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, _ := json.Marshal(payload)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(p.TargetURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("webhook failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return fmt.Sprintf("Webhook sent to %s", p.TargetURL), nil
}
