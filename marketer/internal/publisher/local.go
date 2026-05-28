package publisher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalPublisher struct {
	OutputDir string
}

func init() {
	Register("local", NewLocalPublisher("published"))
}

func NewLocalPublisher(outputDir string) *LocalPublisher {
	if outputDir == "" { outputDir = "published" }
	return &LocalPublisher{OutputDir: outputDir}
}

func (p *LocalPublisher) Publish(topic, content string) (string, error) {
	if err := os.MkdirAll(p.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %v", err)
	}
	filename := fmt.Sprintf("%s_%s.md", time.Now().Format("20060102_150405"), sanitizeTopic(topic))
	path := filepath.Join(p.OutputDir, filename)
	
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeTopic(t string) string {
	return strings.ReplaceAll(strings.ToLower(t), " ", "_")
}
