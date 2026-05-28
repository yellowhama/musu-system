package harvester

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type GitHubFetcher struct{}

type GitHubRepoMetadata struct {
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Stars       int      `json:"stargazers_count"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	HTMLURL     string   `json:"html_url"`
}

func (f *GitHubFetcher) Fetch(repoPath string) (string, string, error) {
	// repoPath format: "owner/repo"
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repo path format, expected owner/repo")
	}
	owner := parts[0]
	repo := parts[1]

	// 1. Fetch Metadata via GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	headers := map[string]string{"User-Agent": USER_AGENT}

	body, _, err := utils.GetWithRetry(apiURL, headers)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch repo metadata: %v", err)
	}

	var meta GitHubRepoMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", "", fmt.Errorf("failed to decode repo metadata: %v", err)
	}

	// 2. Fetch README.md content (Try main then master)
	readmeContent := ""
	branches := []string{"main", "master"}
	for _, branch := range branches {
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/README.md", owner, repo, branch)
		rawBody, status, err := utils.GetWithRetry(rawURL, headers)
		if err == nil && status == 200 {
			readmeContent = string(rawBody)
			break
		}
	}

	if readmeContent == "" {
		return "", "", fmt.Errorf("could not find README.md in main or master branch")
	}

	// Build combined content with some metadata info if needed
	// (Frontmatter is handled by processor, so we just return the body)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", meta.FullName))
	sb.WriteString(fmt.Sprintf("> %s\n\n", meta.Description))
	sb.WriteString(fmt.Sprintf("⭐ Stars: %d | 🌐 Language: %s\n\n", meta.Stars, meta.Language))
	sb.WriteString("---\n\n")
	sb.WriteString(readmeContent)

	return meta.FullName, sb.String(), nil
}
