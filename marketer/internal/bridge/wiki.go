package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"regexp"
	"strings"
)

type WikiBridge struct {
	WikiDir string
}

type KnowledgeSource struct {
	ID      string
	Title   string
	Content string
	Source  string
	Project string
}

type indexEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Source  string   `json:"source"`
	Project string   `json:"project"`
	Path    string   `json:"path"`
	Tags    []string `json:"tags,omitempty"`
	Summary string   `json:"summary,omitempty"`
}

type scoredKnowledgeSource struct {
	KnowledgeSource
	score int
}

func NewWikiBridge(wikiDir string) *WikiBridge {
	return &WikiBridge{WikiDir: wikiDir}
}

func (b *WikiBridge) FindByTopic(topic string) ([]KnowledgeSource, error) {
	if strings.TrimSpace(topic) == "" {
		return nil, nil
	}
	if results, err := b.findByTopicFromIndex(topic); err == nil && len(results) > 0 {
		return results, nil
	}
	return b.findByTopicFromFiles(topic)
}

func (b *WikiBridge) findByTopicFromIndex(topic string) ([]KnowledgeSource, error) {
	indexPath := filepath.Join(b.WikiDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	var entries []indexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(topic))
	tokens := tokenizeQuery(query)
	var scored []scoredKnowledgeSource
	seen := map[string]bool{}
	for _, entry := range entries {
		path := filepath.Join(b.WikiDir, entry.Path)
		body := ""
		if raw, readErr := os.ReadFile(path); readErr == nil {
			body = string(raw)
		}
		score := topicMatchScore(query, tokens, entry.Title, entry.Summary, entry.Path, entry.Tags, body)
		if score == 0 {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		scored = append(scored, scoredKnowledgeSource{
			KnowledgeSource: KnowledgeSource{
				ID:      entry.ID,
				Title:   entry.Title,
				Content: body,
				Source:  entry.Source,
				Project: entry.Project,
			},
			score: score,
		})
	}
	sortScoredSources(scored)
	return flattenScoredSources(scored), nil
}

func (b *WikiBridge) findByTopicFromFiles(topic string) ([]KnowledgeSource, error) {
	query := strings.ToLower(strings.TrimSpace(topic))
	tokens := tokenizeQuery(query)
	var scored []scoredKnowledgeSource
	
	err := filepath.Walk(b.WikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		body := string(data)
		title := filepath.Base(path)
		score := topicMatchScore(query, tokens, title, "", path, nil, body)
		if score > 0 {
			scored = append(scored, scoredKnowledgeSource{
				KnowledgeSource: KnowledgeSource{
				ID:      filepath.Base(path),
				Title:   title,
				Content: body,
				},
				score: score,
			})
		}
		return nil
	})
	
	sortScoredSources(scored)
	return flattenScoredSources(scored), err
}

func flattenScoredSources(scored []scoredKnowledgeSource) []KnowledgeSource {
	results := make([]KnowledgeSource, 0, len(scored))
	for _, item := range scored {
		results = append(results, item.KnowledgeSource)
	}
	return results
}

func sortScoredSources(scored []scoredKnowledgeSource) {
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].Project != scored[j].Project {
			return scored[i].Project < scored[j].Project
		}
		return scored[i].Title < scored[j].Title
	})
}

func topicMatchScore(query string, tokens []string, title string, summary string, path string, tags []string, body string) int {
	score := 0
	lowerTitle := strings.ToLower(title)
	lowerSummary := strings.ToLower(summary)
	lowerPath := strings.ToLower(path)
	lowerBody := strings.ToLower(body)

	if query != "" {
		if strings.Contains(lowerTitle, query) {
			score += 140
		}
		if strings.Contains(lowerSummary, query) {
			score += 90
		}
		if strings.Contains(lowerPath, query) {
			score += 50
		}
		if strings.Contains(lowerBody, query) {
			score += 35
		}
	}

	matchedTokens := 0
	for _, token := range tokens {
		tokenMatched := false
		if strings.Contains(lowerTitle, token) {
			score += 35
			tokenMatched = true
		}
		if strings.Contains(lowerSummary, token) {
			score += 24
			tokenMatched = true
		}
		if strings.Contains(lowerPath, token) {
			score += 12
			tokenMatched = true
		}
		if strings.Contains(lowerBody, token) {
			score += 8
			tokenMatched = true
		}
		for _, tag := range tags {
			if strings.Contains(strings.ToLower(tag), token) {
				score += 40
				tokenMatched = true
			}
		}
		if tokenMatched {
			matchedTokens++
		}
	}

	if len(tokens) > 1 && matchedTokens == len(tokens) {
		score += 45
	}
	if query != "" && score > 0 && strings.EqualFold(strings.TrimSpace(title), strings.TrimSpace(query)) {
		score += 80
	}

	return score
}

var topicSplitPattern = regexp.MustCompile(`[\s\-_./:()]+`)

func tokenizeQuery(query string) []string {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	parts := topicSplitPattern.Split(strings.ToLower(query), -1)
	tokens := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 2 || seen[part] {
			continue
		}
		seen[part] = true
		tokens = append(tokens, part)
	}
	return tokens
}
