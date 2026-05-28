package utils

import "strings"

// DetectSource maps a URL or identifier to its harvester source key.
// It returns "" when the input is not a recognised URL. This is the single
// source of truth for source auto-detection (previously duplicated in cmd and
// agent packages).
func DetectSource(input string) string {
	switch {
	case strings.Contains(input, "youtube.com"), strings.Contains(input, "youtu.be"):
		return "yt"
	case strings.Contains(input, "github.com"):
		return "gh"
	case strings.Contains(input, "arxiv.org"):
		return "arxiv"
	case strings.Contains(input, "huggingface.co"):
		return "hf"
	case strings.Contains(input, "twitter.com"), strings.Contains(input, "x.com"):
		return "x"
	case strings.Contains(input, "reddit.com"):
		return "reddit"
	case strings.HasPrefix(input, "http"):
		return "web"
	default:
		return ""
	}
}
