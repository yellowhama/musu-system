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
	APIKey string
}

func NewSMSPoolProvider(apiKey string) *SMSPoolProvider {
	return &SMSPoolProvider{APIKey: apiKey}
}

func (p *SMSPoolProvider) RequestEmail() (string, error) {
	// For now, we reuse the Mock logic or would use AgentMail/Testmail API here
	return "temp_" + time.Now().Format("150405") + "@mock.com", nil
}

func (p *SMSPoolProvider) RequestPhone(service string) (string, string, error) {
	// SMSPool Purchase API
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
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Success != 1 {
		return "", "", fmt.Errorf("smspool error: %s", result.Message)
	}

	return result.Number, result.OrderID, nil
}

func (p *SMSPoolProvider) CheckSMS(orderID string) (string, error) {
	apiURL := "https://api.smspool.net/sms/check"
	
	// Polling Loop
	for i := 0; i < 30; i++ { // 5 minutes (30 * 10s)
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
		json.Unmarshal(body, &result)

		if result.Status == 3 { // Success
			return result.SMS, nil
		}
		
		fmt.Printf("   ⏳ Waiting for SMS (Attempt %d)... status: %d\n", i+1, result.Status)
		time.Sleep(10 * time.Second)
	}

	return "", fmt.Errorf("timed out waiting for SMS")
}
