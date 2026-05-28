package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/harvester"
	"github.com/yellowhama/musu-crawl-ai/internal/processor"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

// FetchAndSave handles the end-to-end process of fetching content and saving it to the wiki.
func FetchAndSave(source, id, lang string, proc *processor.WikiProcessor, model, baseURL string) (string, string, float64, []string, error) {
	title, text, err := DispatchFetch(source, id, lang)
	if err != nil {
		return "", "", 0, nil, err
	}

	tags := utils.ExtractKeywords(text, 5)
	summary := utils.Summarize(text, 3)

	// AI Intelligence Setup (Telemetry enabled)
	ollama := NewAgentClient(baseURL, model, "", proc.BaseDir, proc.Project)

	// Image Harvesting
	imageDir := filepath.Join(proc.BaseDir, "projects", proc.Project, "images")
	text = utils.DownloadAndRelinkImages(text, imageDir, ollama.DescribeImage)

	reliability := GetReliabilityScore(source)

	// Save
	fname, err := proc.SaveToWikiWithEmbedder(source, id, title, text, tags, summary, reliability, ollama.Embed)
	if err != nil {
		return "", "", 0, nil, err
	}

	return fname, text, reliability, tags, nil
}

func DispatchFetch(source, id, lang string) (string, string, error) {
	source = strings.ToLower(source)
	switch source {
	case "yt", "youtube":
		f := &harvester.YouTubeFetcher{Language: lang}
		return f.Fetch(id)
	case "gh", "github":
		f := &harvester.GitHubFetcher{}
		return f.Fetch(id)
	case "web":
		f := &harvester.WebFetcher{}
		return f.Fetch(id)
	case "arxiv":
		f := &harvester.ArxivFetcher{}
		return f.Fetch(id)
	case "hf", "huggingface":
		f := &harvester.HuggingFaceFetcher{}
		return f.Fetch(id)
	case "x", "twitter":
		f := &harvester.TwitterFetcher{}
		return f.Fetch(id)
	case "reddit":
		f := &harvester.RedditFetcher{}
		return f.Fetch(id)
	default:
		return "", "", fmt.Errorf("unsupported source: %s", source)
	}
}

func GetReliabilityScore(source string) float64 {
	source = strings.ToLower(source)
	switch source {
	case "arxiv", "github", "gh":
		return 0.9
	case "youtube", "yt", "web":
		return 0.7
	case "x", "twitter", "reddit":
		return 0.5
	default:
		return 0.6
	}
}
