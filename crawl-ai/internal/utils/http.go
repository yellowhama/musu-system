package utils

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const (
	MaxRetries     = 3
	InitialBackoff = 1 * time.Second
	MaxBackoff     = 10 * time.Second
)

// GetWithRetry performs an HTTP GET request with exponential backoff retry logic.
func GetWithRetry(url string, headers map[string]string) ([]byte, int, error) {
	var body []byte
	var lastStatus int
	var lastErr error

	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i <= MaxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(math.Pow(2, float64(i-1))) * InitialBackoff
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
			fmt.Printf("⚠️  Retry %d/%d for %s (waiting %v)...\n", i, MaxRetries, url, backoff)
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, 0, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		lastStatus = resp.StatusCode

		// If 200, read body and return
		if resp.StatusCode == 200 {
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = err
				continue
			}
			return body, resp.StatusCode, nil
		}

		resp.Body.Close()

		// If transient error (429 or 5xx), retry. Otherwise, stop and return error.
		if resp.StatusCode != 429 && (resp.StatusCode < 500 || resp.StatusCode > 599) {
			return nil, resp.StatusCode, fmt.Errorf("HTTP error %d", resp.StatusCode)
		}

		lastErr = fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	return nil, lastStatus, fmt.Errorf("failed after %d retries: %v", MaxRetries, lastErr)
}

// PostWithRetry performs an HTTP POST request with exponential backoff retry logic.
func PostWithRetry(url string, headers map[string]string, payload []byte) ([]byte, int, error) {
	var body []byte
	var lastStatus int
	var lastErr error

	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i <= MaxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(math.Pow(2, float64(i-1))) * InitialBackoff
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return nil, 0, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		lastStatus = resp.StatusCode
		if resp.StatusCode == 200 {
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = err
				continue
			}
			return body, resp.StatusCode, nil
		}

		resp.Body.Close()
		if resp.StatusCode != 429 && (resp.StatusCode < 500 || resp.StatusCode > 599) {
			return nil, resp.StatusCode, fmt.Errorf("HTTP error %d", resp.StatusCode)
		}
		lastErr = fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	return nil, lastStatus, fmt.Errorf("failed after %d retries: %v", MaxRetries, lastErr)
}
