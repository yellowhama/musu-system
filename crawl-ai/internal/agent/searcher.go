package agent

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type Searcher struct {
	BaseURL string
}

func (s *Searcher) DiscoverURLs(queries []string, limit int) []string {
	uniqueURLs := make(map[string]bool)
	var results []string

	for _, query := range queries {
		fmt.Printf("🔍 Searching for: %s...\n", query)
		urls := s.searchDuckDuckGo(query)
		for _, u := range urls {
			if !uniqueURLs[u] {
				uniqueURLs[u] = true
				results = append(results, u)
			}
			if len(results) >= limit {
				break
			}
		}
		if len(results) >= limit {
			break
		}
		// Small delay to be polite
		time.Sleep(1 * time.Second)
	}

	return results
}

func (s *Searcher) searchDuckDuckGo(query string) []string {
	baseURL := strings.TrimSpace(s.BaseURL)
	searchURL := ""
	if baseURL == "" {
		searchURL = fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	} else {
		searchURL = fmt.Sprintf("%s?q=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(query))
	}

	body, _, err := utils.GetWithRetry(searchURL, nil)
	if err != nil {
		fmt.Printf("⚠️  Search failed: %v\n", err)
		return nil
	}

	htmlStr := string(body)
	// Regex to extract result links from DDG HTML
	// This is a basic scraper, in a real scenario consider a proper API
	re := regexp.MustCompile(`class="result__url".*?href="(.*?)"`)
	matches := re.FindAllStringSubmatch(htmlStr, -1)

	var urls []string
	for _, m := range matches {
		u := m[1]
		// Clean up the URL if it's a proxy link
		if strings.Contains(u, "uddg=") {
			parsed, _ := url.Parse(u)
			u = parsed.Query().Get("uddg")
		}
		if u != "" && !strings.Contains(u, "duckduckgo.com") {
			urls = append(urls, u)
		}
	}

	return urls
}
