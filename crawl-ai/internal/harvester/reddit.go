package harvester

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

type RedditFetcher struct{}

type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Kind string `json:"kind"`
			Data struct {
				ID        string  `json:"id"`
				Title     string  `json:"title"`
				Selftext  string  `json:"selftext"`
				Author    string  `json:"author"`
				Subreddit string  `json:"subreddit"`
				Permalink string  `json:"permalink"`
				URL       string  `json:"url"`
				Score     int     `json:"score"`
				NumComms  int     `json:"num_comments"`
				Created   float64 `json:"created_utc"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type RedditTrend struct {
	Title     string
	URL       string
	Score     int
	Comments  int
	Subreddit string
}

func (f *RedditFetcher) Fetch(url string) (string, string, error) {
	jsonURL := strings.TrimRight(url, "/")
	if !strings.HasSuffix(jsonURL, ".json") {
		jsonURL += ".json"
	}

	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 musu-crawl-ai/0.7.2",
	}

	body, _, err := utils.GetWithRetry(jsonURL, headers)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch reddit data: %v", err)
	}

	if strings.HasPrefix(string(body), "[") {
		var resp []RedditResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", "", fmt.Errorf("failed to decode reddit array JSON: %v", err)
		}
		if len(resp) > 0 {
			return f.formatPost(resp[0])
		}
	} else {
		var resp RedditResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", "", fmt.Errorf("failed to decode reddit object JSON: %v", err)
		}
		return f.formatPost(resp)
	}

	return "", "", fmt.Errorf("no content found in reddit response")
}

func (f *RedditFetcher) Spot(subreddit string, limit int) ([]RedditTrend, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, limit)
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 musu-crawl-ai/0.7.2",
	}

	body, _, err := utils.GetWithRetry(url, headers)
	if err != nil {
		return nil, err
	}

	var resp RedditResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var trends []RedditTrend
	for _, child := range resp.Data.Children {
		d := child.Data
		trends = append(trends, RedditTrend{
			Title:     d.Title,
			URL:       "https://reddit.com" + d.Permalink,
			Score:     d.Score,
			Comments:  d.NumComms,
			Subreddit: d.Subreddit,
		})
	}
	return trends, nil
}

func (f *RedditFetcher) formatPost(resp RedditResponse) (string, string, error) {
	if len(resp.Data.Children) == 0 {
		return "", "", fmt.Errorf("reddit response contains no data")
	}

	post := resp.Data.Children[0].Data
	title := post.Title

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", post.Title))
	sb.WriteString(fmt.Sprintf("**Author:** u/%s | **Subreddit:** r/%s\n\n", post.Author, post.Subreddit))
	sb.WriteString("---\n\n")

	if post.Selftext != "" {
		sb.WriteString(post.Selftext)
	} else if post.URL != "" {
		sb.WriteString(fmt.Sprintf("External Link: %s", post.URL))
	}

	return title, sb.String(), nil
}
