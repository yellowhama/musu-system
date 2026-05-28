package harvester

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

const USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type YouTubeFetcher struct {
	Language string
}

type CaptionTrack struct {
	BaseURL      string `json:"baseUrl"`
	LanguageCode string `json:"languageCode"`
}

type PlayerResponse struct {
	Captions struct {
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []CaptionTrack `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
	VideoDetails struct {
		Title string `json:"title"`
	} `json:"videoDetails"`
}

type TranscriptResponse struct {
	Events []struct {
		Segs []struct {
			Utf8 string `json:"utf8"`
		} `json:"segs"`
	} `json:"events"`
}

func (f *YouTubeFetcher) Fetch(videoID string) (string, string, error) {
	// 1. Try public watch page
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	headers := map[string]string{"User-Agent": USER_AGENT}

	body, _, err := utils.GetWithRetry(watchURL, headers)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch watch page: %v", err)
	}

	htmlStr := string(body)

	re := regexp.MustCompile(`ytInitialPlayerResponse\s*=\s*({.+?});`)
	matches := re.FindStringSubmatch(htmlStr)

	var pr PlayerResponse
	apiKey := ""
	if len(matches) >= 2 {
		if uerr := json.Unmarshal([]byte(matches[1]), &pr); uerr != nil {
			fmt.Fprintf(os.Stderr, "warn: parse youtube player response: %v\n", uerr)
		}
	}

	reKey := regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	if m := reKey.FindStringSubmatch(htmlStr); len(m) >= 2 {
		apiKey = m[1]
	}

	tracks := pr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
	isGated := false
	for _, t := range tracks {
		if strings.Contains(t.BaseURL, "exp=xpe") || strings.Contains(t.BaseURL, "exp=xpv") {
			isGated = true
			break
		}
	}

	// 2. Fallback to Innertube if gated or no tracks
	if isGated || len(tracks) == 0 {
		if apiKey != "" {
			ipr, err := f.fetchInnertubePlayer(videoID, apiKey)
			if err == nil && len(ipr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks) > 0 {
				tracks = ipr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
				if pr.VideoDetails.Title == "" {
					pr.VideoDetails.Title = ipr.VideoDetails.Title
				}
			}
		}
	}

	if len(tracks) == 0 {
		return "", "", fmt.Errorf("no captions found for %s", videoID)
	}

	// Select track
	var target *CaptionTrack
	for _, t := range tracks {
		if t.LanguageCode == f.Language {
			target = &t
			break
		}
	}
	if target == nil {
		target = &tracks[0]
	}

	// 3. Download
	text, err := f.downloadTranscript(target.BaseURL)
	if err != nil {
		return "", "", err
	}

	title := pr.VideoDetails.Title
	if title == "" {
		title = videoID
	}

	return title, text, nil
}

func (f *YouTubeFetcher) fetchInnertubePlayer(videoID, apiKey string) (*PlayerResponse, error) {
	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "ANDROID",
				"clientVersion": "21.02.35",
				"userAgent":     "com.google.android.youtube/21.02.35 (Linux; U; Android 11) gzip",
			},
		},
		"videoId": videoID,
	}
	body, _ := json.Marshal(payload)
	postURL := fmt.Sprintf("https://www.youtube.com/youtubei/v1/player?key=%s", apiKey)

	headers := map[string]string{
		"Content-Type":             "application/json",
		"X-YouTube-Client-Name":    "3",
		"X-YouTube-Client-Version": "21.02.35",
	}

	resBody, _, err := utils.PostWithRetry(postURL, headers, body)
	if err != nil {
		return nil, err
	}

	var pr PlayerResponse
	if err := json.Unmarshal(resBody, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

func (f *YouTubeFetcher) downloadTranscript(baseURL string) (string, error) {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("fmt", "json3")
	u.RawQuery = q.Encode()

	headers := map[string]string{"User-Agent": USER_AGENT}
	body, _, err := utils.GetWithRetry(u.String(), headers)
	if err != nil {
		return "", err
	}

	var tr TranscriptResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, event := range tr.Events {
		for _, seg := range event.Segs {
			sb.WriteString(seg.Utf8)
		}
		sb.WriteString(" ")
	}
	return strings.TrimSpace(sb.String()), nil
}
