package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
)

type Compiler struct {
	Client  *AgentClient
	Index   bleve.Index
	WikiDir string
}

type Relationship struct {
	TargetTitle string
	TargetPath  string
	Explanation string
}

func NewCompiler(client *AgentClient, wikiDir string) (*Compiler, error) {
	blevePath := filepath.Join(wikiDir, "musu.bleve")
	index, err := bleve.Open(blevePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bleve: %v", err)
	}

	return &Compiler{
		Client:  client,
		Index:   index,
		WikiDir: wikiDir,
	}, nil
}

func (c *Compiler) Close() {
	if c.Index != nil {
		c.Index.Close()
	}
}

func (c *Compiler) CompileDocument(docPath string, content string, tags []string, summary string) (string, error) {
	// 1. Find related documents using Bleve
	// We'll search using tags and the first sentence of the summary
	searchQuery := strings.Join(tags, " ")
	if len(summary) > 50 {
		searchQuery += " " + summary[:50]
	}

	query := bleve.NewQueryStringQuery(searchQuery)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"title", "source", "id", "summary", "path"}
	searchRequest.Size = 4 // Get top 4 related docs

	searchResults, err := c.Index.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}

	var relationships []Relationship
	for _, hit := range searchResults.Hits {
		// Don't link to itself
		relPath := hit.Fields["path"].(string)
		fullRelPath := filepath.Join(c.WikiDir, relPath)
		absDocPath, _ := filepath.Abs(docPath)
		absRelPath, _ := filepath.Abs(fullRelPath)
		if absDocPath == absRelPath {
			continue
		}

		relTitle := hit.Fields["title"].(string)
		relSummary := ""
		if s, ok := hit.Fields["summary"]; ok {
			relSummary = s.(string)
		}

		// 2. Ask Ollama to explain the relationship
		explanation, err := c.explainRelationship(summary, relTitle, relSummary)
		if err != nil {
			fmt.Printf("   ⚠️  Failed to explain relationship with %s: %v\n", relTitle, err)
			continue
		}

		relationships = append(relationships, Relationship{
			TargetTitle: relTitle,
			TargetPath:  relPath,
			Explanation: explanation,
		})
	}

	if len(relationships) == 0 {
		return "", nil
	}

	// 3. Format the "Related Knowledge" section
	var sb strings.Builder
	sb.WriteString("\n\n---\n\n## 🧠 Related Knowledge (Auto-Compiled)\n\n")
	for _, rel := range relationships {
		// Using Obsidian style links [[path|Title]] or standard markdown
		// We'll stick to standard markdown for compatibility
		sb.WriteString(fmt.Sprintf("*   **[%s](../%s)**: %s\n", rel.TargetTitle, rel.TargetPath, rel.Explanation))
	}

	return sb.String(), nil
}

func (c *Compiler) explainRelationship(sourceSummary, targetTitle, targetSummary string) (string, error) {
	prompt := fmt.Sprintf(`You are a Knowledge Graph Compiler. 
Determine how Document A is related to Document B.

Document A Summary: %s

Document B Title: %s
Document B Summary: %s

Task: Briefly explain the connection or shared concepts between these two documents in one concise sentence. 
Focus on why a researcher reading A should care about B.
Response must be just the sentence, no preamble.`, sourceSummary, targetTitle, targetSummary)

	resp, err := c.Client.Ask(prompt, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

func (c *Compiler) UpdateDocument(path string, section string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(section)
	return err
}
