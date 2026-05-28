package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)


var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update musu-marketer to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Checking for updates...")

		latest, url, err := GetLatestRelease("yellowhama", "musu-marketer")
		if err != nil {
			fmt.Printf("❌ Failed to check for updates: %v\n", err)
			return
		}

		if latest == Version {
			fmt.Printf("✅ You are already on the latest version (%s).\n", Version)
			return
		}

		fmt.Printf("🆕 New version available: %s (current: %s)\n", latest, Version)
		fmt.Print("⏳ Downloading and updating... ")

		err = doUpdate(url)
		if err != nil {
			fmt.Printf("\n❌ Update failed: %v\n", err)
			return
		}

		fmt.Println("✅ Done!")
		fmt.Printf("🎉 musu-marketer has been updated to %s.\n", latest)
	},
}

func GetLatestRelease(owner, repo string) (string, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	suffix := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}

	downloadURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, suffix) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no matching binary found")
	}

	return release.TagName, downloadURL, nil
}

func doUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	oldPath := exePath + ".old"
	os.Remove(oldPath) 
	err = os.Rename(exePath, oldPath)
	if err != nil {
		return fmt.Errorf("failed to rename current binary: %v", err)
	}

	newFile, err := os.OpenFile(exePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		os.Rename(oldPath, exePath) 
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, resp.Body)
	if err != nil {
		os.Rename(oldPath, exePath) 
		return err
	}

	if runtime.GOOS != "windows" {
		os.Remove(oldPath)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
