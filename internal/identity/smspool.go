package identity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SMSPoolProvider struct {
	APIKey   string
	EmailKey string // Optional: API Key for email service if used
}

func NewSMSPoolProvider(apiKey string) *SMSPoolProvider {
	return &SMSPoolProvider{APIKey: apiKey}
}

// RequestEmail now generates a more realistic 'private' style email address.
// In a full implementation, this would link to Testmail.app or Mailosaur APIs.
func (p *SMSPoolProvider) RequestEmail() (string, error) {
	// 2025 Strategy: Use a more realistic naming pattern for stealth
	timestamp := time.Now().Format("20060102")
	randomSuffix := fmt.Sprintf("%d", time.Now().UnixNano()%10000)
	
	// Defaulting to a generic pattern that agents can easily identify and replace with real API-backed emails
	return fmt.Sprintf("citizen.%s.%s@agentmail.com", timestamp, randomSuffix), nil
}

func (p *SMSPoolProvider) RequestPhone(service string) (string, string, error) {
	apiURL := "https://api.smspool.net/purchase/sms"
	data := url.Values{}
	data.Set("key", p.APIKey)
	data.Set("country", "1") // US
	data.Set("service", service)

	resp, err := http.PostForm(apiURL, data)
	if err != nil { return "", "", err }
	defer resp.Body.Close()

	var result struct {
		Success int    `json:"success"`
		Number  string `json:"number"`
		OrderID string `json:"orderid"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode smspool response: %v", err)
	}

	if result.Success != 1 {
		return "", "", fmt.Errorf("smspool error: %s", result.Message)
	}

	return result.Number, result.OrderID, nil
}

func (p *SMSPoolProvider) CheckSMS(orderID string) (string, error) {
	apiURL := "https://api.smspool.net/sms/check"
	
	for i := 0; i < 30; i++ { 
		data := url.Values{}
		data.Set("key", p.APIKey)
		data.Set("orderid", orderID)

		resp, err := http.PostForm(apiURL, data)
		if err != nil { return "", err }
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Status int    `json:"status"`
			SMS    string `json:"sms"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to unmarshal sms check: %v", err)
		}

		if result.Status == 3 { 
			return result.SMS, nil
		}
		
		fmt.Printf("[SYSTEM] Waiting for SMS (Attempt %d)... status: %d\n", i+1, result.Status)
		time.Sleep(10 * time.Second)
	}

	return "", fmt.Errorf("timed out waiting for SMS")
}
