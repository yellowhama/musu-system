package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// DownloadAndRelinkImages finds image links in markdown, downloads them to local storage,
// and returns the modified markdown with relative local paths.
// It also takes an optional visionFunc to generate captions for the images.
func DownloadAndRelinkImages(markdown string, imageDir string, visionFunc func(string) (string, error)) string {
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		// Can't stage images locally; leave the markdown (and its remote links) untouched.
		return markdown
	}

	// Regex to find markdown image links: ![alt](url)
	re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

	client := &http.Client{Timeout: 30 * time.Second}

	return re.ReplaceAllStringFunc(markdown, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		altText := submatches[1]
		imgURL := submatches[2]

		// Skip if already local or non-http
		if !regexp.MustCompile(`^https?://`).MatchString(imgURL) {
			return match
		}

		// 1. Generate unique filename based on URL hash
		hash := sha256.Sum256([]byte(imgURL))
		ext := filepath.Ext(imgURL)
		if ext == "" {
			ext = ".png" // Default
		}
		fileName := fmt.Sprintf("%x%s", hash[:8], ext)
		localPath := filepath.Join(imageDir, fileName)

		// 2. Download if not already exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			fmt.Printf("[SYSTEM] Downloading image: %s\n", imgURL)
			resp, err := client.Get(imgURL)
			if err != nil {
				return match // Fallback to original
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				out, err := os.Create(localPath)
				if err == nil {
					io.Copy(out, resp.Body)
					out.Close()
				}
			}
		}

		// 3. Vision Analysis (Optional)
		caption := ""
		if visionFunc != nil {
			fmt.Printf("[VISION] Analyzing: %s...\n", fileName)
			if desc, err := visionFunc(localPath); err == nil {
				caption = desc
			}
		}

		// 4. Return relinked markdown
		// Path should be relative to the markdown file (which is in {project}/{source}/)
		// Images are in {project}/images/
		if caption != "" {
			return fmt.Sprintf("![%s](../images/%s)\n> **Vision AI:** %s", altText, fileName, caption)
		}
		return fmt.Sprintf("![%s](../images/%s)", altText, fileName)
	})
}
