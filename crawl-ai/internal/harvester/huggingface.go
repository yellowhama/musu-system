package harvester

import (
	"fmt"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type HuggingFaceFetcher struct{}

func (f *HuggingFaceFetcher) Fetch(modelID string) (string, string, error) {
	// 1. Fetch README.md (Model Card)
	// HF Raw URL pattern: https://huggingface.co/owner/model/raw/main/README.md
	rawURL := fmt.Sprintf("https://huggingface.co/%s/raw/main/README.md", modelID)
	headers := map[string]string{"User-Agent": USER_AGENT}

	body, _, err := utils.GetWithRetry(rawURL, headers)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch model card: %v", err)
	}

	readmeContent := string(body)

	// 2. Metadata (Using model ID as title for now, or parsing from README YAML if present)
	title := modelID
	// Extract title from YAML frontmatter if possible
	if strings.HasPrefix(readmeContent, "---") {
		lines := strings.Split(readmeContent, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "title:") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
				title = strings.Trim(title, "\"'")
				break
			}
		}
	}

	return title, readmeContent, nil
}
