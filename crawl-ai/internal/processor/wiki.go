package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"gopkg.in/yaml.v3"
)

type WikiProcessor struct {
	BaseDir string
	Project string
	mu      sync.Mutex   // Protects file operations and indexing
	vstore  *VectorStore // Optimized: Persistent in-memory vector cache
}

type IndexEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Source      string   `json:"source"`
	Project     string   `json:"project"`
	Reliability float64  `json:"reliability"`
	Path        string   `json:"path"`
	Date        string   `json:"date"`
	Tags        []string `json:"tags,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Content     string   `json:"-"` // Not in JSON but indexed in Bleve
}

func NewWikiProcessor(baseDir string, project string) *WikiProcessor {
	if project == "" {
		project = "default"
	}
	p := &WikiProcessor{
		BaseDir: baseDir,
		Project: project,
		vstore:  NewVectorStore(),
	}
	
	// Pre-load vectors if they exist
	vectorFile := filepath.Join(baseDir, "musu.vectors.json")
	p.vstore.Load(vectorFile)
	
	return p
}

// SaveToWiki is now a thin wrapper for SaveToWikiWithEmbedder with nil embedder
func (p *WikiProcessor) SaveToWiki(source, rawID, title, content string, tags []string, summary string, reliability float64) (string, error) {
	return p.SaveToWikiWithEmbedder(source, rawID, title, content, tags, summary, reliability, nil)
}

func (p *WikiProcessor) SaveToWikiWithEmbedder(source, rawID, title, content string, tags []string, summary string, reliability float64, embedder func(string) ([]float64, error)) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	sourceDir := p.GetSourceDir(source)
	safeID := p.NormalizeID(source, rawID)

	dir := filepath.Join(p.BaseDir, "projects", p.Project, sourceDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	safeTitle := sanitizeFilename(title)
	filename := fmt.Sprintf("%s_%s.md", safeID, safeTitle)
	if len(filename) > 200 {
		filename = filename[:200] + ".md"
	}
	path := filepath.Join(dir, filename)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	sb.WriteString(fmt.Sprintf("source: %s\n", source))
	sb.WriteString(fmt.Sprintf("project: %s\n", p.Project))
	sb.WriteString(fmt.Sprintf("reliability: %.2f\n", reliability))
	sb.WriteString(fmt.Sprintf("id: %s\n", safeID))
	sb.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("summary: %q\n", summary))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(content)

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	// ⚡ Optimized Live Sync (Incremental Indexing)
	relPath, _ := filepath.Rel(p.BaseDir, path)
	entry := IndexEntry{
		ID:          safeID,
		Title:       title,
		Source:      source,
		Project:     p.Project,
		Reliability: reliability,
		Path:        relPath,
		Date:        time.Now().Format("2006-01-02"),
		Tags:        tags,
		Summary:     summary,
		Content:     content,
	}

	p.indexSingleDocumentLocked(entry, embedder)

	return filename, nil
}

func (p *WikiProcessor) indexSingleDocumentLocked(entry IndexEntry, embedder func(string) ([]float64, error)) error {
	blevePath := filepath.Join(p.BaseDir, "musu.bleve")
	indexFile := filepath.Join(p.BaseDir, "index.json")
	vectorFile := filepath.Join(p.BaseDir, "musu.vectors.json")

	// 1. Bleve Index
	var idx bleve.Index
	var err error
	if _, statErr := os.Stat(blevePath); os.IsNotExist(statErr) {
		mapping := bleve.NewIndexMapping()
		idx, err = bleve.New(blevePath, mapping)
	} else {
		idx, err = bleve.Open(blevePath)
	}
	if err == nil {
		idx.Index(entry.ID, entry)
		idx.Close()
	}

	// 2. Vector Index (Optimized: USES p.vstore CACHE)
	if embedder != nil {
		textToEmbed := entry.Summary
		if textToEmbed == "" {
			textToEmbed = entry.Title
		}
		if vec, err := embedder(textToEmbed); err == nil {
			p.vstore.Embeddings[entry.ID] = vec
			p.vstore.Save(vectorFile)
		}
	}

	// 3. JSON Index
	var entries []IndexEntry
	if data, err := os.ReadFile(indexFile); err == nil {
		if uerr := json.Unmarshal(data, &entries); uerr != nil {
			fmt.Fprintf(os.Stderr, "warn: parse %s: %v\n", indexFile, uerr)
		}
	}
	
	found := false
	for i, e := range entries {
		if e.ID == entry.ID {
			entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, entry)
	}

	jsonData, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(indexFile, jsonData, 0644); err != nil {
		return fmt.Errorf("write index %s: %v", indexFile, err)
	}

	return nil
}

func (p *WikiProcessor) GetSourceDir(source string) string {
	source = strings.ToLower(source)
	switch source {
	case "yt", "youtube":
		return "youtube"
	case "gh", "github":
		return "github"
	case "arxiv":
		return "papers"
	case "hf", "huggingface":
		return "huggingface"
	case "x", "twitter":
		return "twitter"
	default:
		return source
	}
}

func (p *WikiProcessor) NormalizeID(source, id string) string {
	source = strings.ToLower(source)
	if source == "gh" || source == "github" || source == "hf" || source == "huggingface" {
		return strings.ReplaceAll(id, "/", "_")
	}
	if source == "web" || source == "reddit" || source == "x" || source == "twitter" {
		safe := strings.ReplaceAll(id, "https://", "")
		safe = strings.ReplaceAll(safe, "http://", "")
		safe = strings.ReplaceAll(safe, "/", "_")
		safe = strings.ReplaceAll(safe, ":", "_")
		safe = strings.ReplaceAll(safe, "?", "_")
		safe = strings.ReplaceAll(safe, "&", "_")
		safe = strings.ReplaceAll(safe, "=", "_")
		if len(safe) > 100 {
			return safe[:100]
		}
		return safe
	}
	return id
}

func (p *WikiProcessor) UpdateIndex() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.updateIndexWithEmbedderLocked(nil)
}

func (p *WikiProcessor) UpdateIndexWithEmbedder(embedder func(string) ([]float64, error)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.updateIndexWithEmbedderLocked(embedder)
}

func (p *WikiProcessor) updateIndexWithEmbedderLocked(embedder func(string) ([]float64, error)) error {
	readmeFile := filepath.Join(p.BaseDir, "README.md")
	indexFile := filepath.Join(p.BaseDir, "index.json")
	blevePath := filepath.Join(p.BaseDir, "musu.bleve")
	vectorFile := filepath.Join(p.BaseDir, "musu.vectors.json")

	// Bleve setup
	var index bleve.Index
	var err error
	if _, statErr := os.Stat(blevePath); os.IsNotExist(statErr) {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(blevePath, mapping)
	} else {
		index, err = bleve.Open(blevePath)
	}
	if err != nil {
		return fmt.Errorf("failed to open bleve: %v", err)
	}
	defer index.Close()

	var sb strings.Builder
	sb.WriteString("# Musu Crawl Wiki Index\n\n")
	sb.WriteString("Automated knowledge repository.\n\n")

	var entries []IndexEntry
	projectsDir := filepath.Join(p.BaseDir, "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(projectsDir, 0755); mkErr != nil {
			return fmt.Errorf("failed to create projects dir: %v", mkErr)
		}
	}

	err = filepath.Walk(p.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Base(path) == "README.md" || filepath.Base(path) == "index.json" || strings.HasSuffix(path, ".bleve") || filepath.Base(path) == "musu.vectors.json" {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		rel, _ := filepath.Rel(p.BaseDir, path)
		link := strings.ReplaceAll(rel, "\\", "/")

		entry, docContent, _ := p.ParseFrontmatterWithContent(path, link)
		if entry != nil {
			entry.Content = docContent
			entries = append(entries, *entry)
			index.Index(entry.ID, entry)

			if embedder != nil {
				if _, exists := p.vstore.Embeddings[entry.ID]; !exists {
					fmt.Printf("🧠 Generating embedding for %s...\n", entry.ID)
					textToEmbed := entry.Summary
					if textToEmbed == "" { textToEmbed = entry.Title }
					vec, err := embedder(textToEmbed)
					if err == nil { p.vstore.Embeddings[entry.ID] = vec }
				}
			}
			sb.WriteString(fmt.Sprintf("* [%s](%s) [%s] (%s)\n", entry.Title, entry.Path, entry.Project, entry.Source))
		}
		return nil
	})

	if err != nil {
		return err
	}

	if err := os.WriteFile(readmeFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write readme %s: %v", readmeFile, err)
	}
	jsonData, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(indexFile, jsonData, 0644); err != nil {
		return fmt.Errorf("write index %s: %v", indexFile, err)
	}
	return p.vstore.Save(vectorFile)
}

func (p *WikiProcessor) ParseFrontmatterWithContent(path string, relPath string) (*IndexEntry, string, error) {
	data, err := os.ReadFile(path)
	if err != nil { return nil, "", err }
	content := string(data)
	if !strings.HasPrefix(content, "---") { return nil, "", fmt.Errorf("no frontmatter") }
	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 { return nil, "", fmt.Errorf("invalid frontmatter") }
	yamlPart := content[3 : endIdx+3]
	docContent := strings.TrimSpace(content[endIdx+6:])
	var meta struct {
		Title       string   `yaml:"title"`
		Source      string   `yaml:"source"`
		Project     string   `yaml:"project"`
		Reliability float64  `yaml:"reliability"`
		ID          string   `yaml:"id"`
		Date        string   `yaml:"date"`
		Tags        []string `yaml:"tags"`
		Summary     string   `yaml:"summary"`
	}
	if err := yaml.Unmarshal([]byte(yamlPart), &meta); err != nil { return nil, "", err }
	if meta.Project == "" { meta.Project = "default" }
	if meta.Reliability == 0 { meta.Reliability = 0.7 }
	return &IndexEntry{
		ID:          meta.ID,
		Title:       meta.Title,
		Source:      meta.Source,
		Project:     meta.Project,
		Reliability: meta.Reliability,
		Path:        relPath,
		Date:        meta.Date,
		Tags:        meta.Tags,
		Summary:     meta.Summary,
	}, docContent, nil
}

func sanitizeFilename(s string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	return re.ReplaceAllString(s, "_")
}
