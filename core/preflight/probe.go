// Package preflight is the shared doctor / preflight building block for the
// Musu CLIs. Most of `doctor` logic is repo-specific (wiki vs mailbox vs
// topic), but the AI endpoint reachability probe is byte-identical across the
// three CLIs, so it lives here once.
package preflight

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Probe checks that the OpenAI-compatible endpoint at baseURL responds to
// GET /models within 3 seconds with a 2xx status. A trailing slash on baseURL
// is tolerated, and a whitespace-only baseURL returns a clear error.
func Probe(baseURL string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("empty ai-url")
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status %s from %s/models", resp.Status, baseURL)
}
