package harvester

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/go-shiori/go-readability"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type WebFetcher struct{}

func (f *WebFetcher) Fetch(targetURL string) (string, string, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %v", err)
	}

	body, _, err := utils.GetWithRetry(targetURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %v", err)
	}

	// 1. Extract main content using readability
	article, err := readability.FromReader(bytes.NewReader(body), parsedURL)

	var finalMD string
	var title string

	if err == nil && article.Content != "" {
		title = article.Title
		markdown, mdErr := htmltomarkdown.ConvertString(article.Content)
		if mdErr == nil {
			finalMD = markdown
		} else {
			finalMD = article.TextContent
		}
	} else {
		// 2. Fallback: Convert raw HTML body if readability fails
		title = "Raw Extraction: " + targetURL
		markdown, mdErr := htmltomarkdown.ConvertString(string(body))
		if mdErr == nil {
			finalMD = markdown
		} else {
			return "", "", fmt.Errorf("readability and fallback conversion failed")
		}
	}

	// 3. Post-conversion cleanup
	finalMD = f.cleanupMarkdown(finalMD)

	return title, finalMD, nil
}

func (f *WebFetcher) cleanupMarkdown(md string) string {
	// Remove common noisy artifacts
	// 1. Remove empty image tags: ![]() or ![](...) with no meaningful content
	reImg := regexp.MustCompile(`!\[\]\(.*?\)\s*`)
	md = reImg.ReplaceAllString(md, "")

	// 2. Reduce multiple empty lines to maximum 2
	reLines := regexp.MustCompile(`\n{3,}`)
	md = reLines.ReplaceAllString(md, "\n\n")

	// 3. Remove typical "Skip to content" or "Cookie notice" lines if they leaked in
	lines := strings.Split(md, "\n")
	var filtered []string
	noise := []string{"skip to content", "cookie policy", "accept all cookies", "privacy policy"}
	for _, line := range lines {
		lower := strings.ToLower(line)
		isNoise := false
		for _, n := range noise {
			if strings.Contains(lower, n) && len(line) < 50 {
				isNoise = true
				break
			}
		}
		if !isNoise {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}
