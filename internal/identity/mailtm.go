package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MailTMProvider struct {
	BaseURL    string
	httpClient *http.Client
}

func NewMailTMProvider() *MailTMProvider {
	return &MailTMProvider{
		BaseURL: "https://api.mail.tm",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDomain gets an available domain from Mail.tm
func (p *MailTMProvider) GetDomain() (string, error) {
	resp, err := p.httpClient.Get(p.BaseURL + "/domains")
	if err != nil { return "", err }
	defer resp.Body.Close()

	var result struct {
		HydraMember []struct {
			Domain string `json:"domain"`
		} `json:"hydra:member"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.HydraMember) == 0 {
		return "", fmt.Errorf("no domains available")
	}
	return result.HydraMember[0].Domain, nil
}

// RequestEmail creates a REAL email account and returns the address.
func (p *MailTMProvider) CreateRealEmail(address, password string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"address":  address,
		"password": password,
	})

	resp, err := p.httpClient.Post(p.BaseURL+"/accounts", "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create email: %s", string(body))
	}

	return address, nil
}

// CheckEmail for verification codes
func (p *MailTMProvider) GetLatestMessage(token string) (string, error) {
	req, _ := http.NewRequest("GET", p.BaseURL+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	var result struct {
		HydraMember []struct {
			Intro string `json:"intro"`
		} `json:"hydra:member"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.HydraMember) == 0 {
		return "", fmt.Errorf("no messages yet")
	}

	return result.HydraMember[0].Intro, nil
}
