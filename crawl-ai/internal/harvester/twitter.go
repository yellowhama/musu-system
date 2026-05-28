package harvester

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type TwitterFetcher struct{}

type TwitterResult struct {
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
	User      struct {
		Name       string `json:"name"`
		ScreenName string `json:"screen_name"`
	} `json:"user"`
}

func (f *TwitterFetcher) Fetch(input string) (string, string, error) {
	// Extract tweet ID from URL or raw ID
	tweetID := f.extractTweetID(input)
	if tweetID == "" {
		return "", "", fmt.Errorf("invalid tweet ID or URL: %s", input)
	}

	// Use Twitter Syndication API (Public, unauthenticated)
	apiURL := fmt.Sprintf("https://cdn.syndication.twimg.com/tweet-result?id=%s&token=x", tweetID)
	headers := map[string]string{"User-Agent": USER_AGENT}

	body, status, err := utils.GetWithRetry(apiURL, headers)
	if err != nil {
		if status == 404 {
			// Fallback to OEmbed if syndication fails
			return f.fetchViaOEmbed(tweetID)
		}
		return "", "", fmt.Errorf("failed to fetch tweet: %v", err)
	}

	var result TwitterResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to decode tweet JSON: %v", err)
	}

	title := fmt.Sprintf("Tweet by @%s", result.User.ScreenName)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## @%s (%s)\n\n", result.User.ScreenName, result.User.Name))
	sb.WriteString(result.Text)
	sb.WriteString("\n\n---\n")
	sb.WriteString(fmt.Sprintf("*Posted on: %s*", result.CreatedAt))

	return title, sb.String(), nil
}

func (f *TwitterFetcher) fetchViaOEmbed(tweetID string) (string, string, error) {
	oembedURL := fmt.Sprintf("https://publish.twitter.com/oembed?url=https://twitter.com/i/status/%s", tweetID)
	body, _, err := utils.GetWithRetry(oembedURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("oembed fallback failed: %v", err)
	}

	var result struct {
		AuthorName string `json:"author_name"`
		HTML       string `json:"html"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to decode oembed JSON: %v", err)
	}

	// Basic regex cleanup for oembed HTML (it contains a <blockquote>)
	re := regexp.MustCompile(`<p.*?>(.*?)<\/p>`)
	match := re.FindStringSubmatch(result.HTML)
	text := ""
	if len(match) >= 2 {
		text = match[1]
		// Remove HTML tags
		text = regexp.MustCompile(`<.*?>`).ReplaceAllString(text, "")
	} else {
		text = "Content unavailable in fallback mode."
	}

	title := fmt.Sprintf("Tweet by %s", result.AuthorName)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s (via OEmbed Fallback)\n\n", result.AuthorName))
	sb.WriteString(text)
	sb.WriteString("\n\n---\n")
	sb.WriteString("*Status: Fetched via public OEmbed endpoint.*")

	return title, sb.String(), nil
}

func (f *TwitterFetcher) extractTweetID(input string) string {
	input = strings.TrimSpace(input)
	// Match 10+ digit ID
	if regexp.MustCompile(`^\d{10,}$`).MatchString(input) {
		return input
	}
	// Extract from URL: twitter.com/user/status/123... or x.com/user/status/123...
	re := regexp.MustCompile(`/status/(\d+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
