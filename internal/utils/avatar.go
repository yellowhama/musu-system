package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GenerateAvatar downloads a deterministic avatar from DiceBear based on a seed.
func GenerateAvatar(seed string, outputDir string) (string, error) {
	os.MkdirAll(outputDir, 0755)
	
	// Use the 'adventurer' style from DiceBear
	url := fmt.Sprintf("https://api.dicebear.com/7.x/adventurer/png?seed=%s", seed)
	fileName := fmt.Sprintf("avatar_%s_%d.png", seed, time.Now().Unix())
	localPath := filepath.Join(outputDir, fileName)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("dicebear returned status: %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return localPath, err
}
