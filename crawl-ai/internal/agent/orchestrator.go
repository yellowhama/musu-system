package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/yellowhama/musu-crawl-ai/internal/processor"
	"github.com/yellowhama/musu-crawl-ai/internal/utils"
)

// Orchestrator provides high-level actions for both CLI and MCP.
type Orchestrator struct {
	WikiDir string
}

func NewOrchestrator(wikiDir string) *Orchestrator {
	if wikiDir == "" { wikiDir = "./wiki" }
	return &Orchestrator{WikiDir: wikiDir}
}

func (o *Orchestrator) FetchAction(source, id, project, lang, model string) (string, error) {
	conf, _ := utils.LoadConfig(project)
	if model == "" { model = conf.AIModel }
	
	proc := processor.NewWikiProcessor(o.WikiDir, project)
	_, text, _, _, err := FetchAndSave(source, id, lang, proc, model, conf.AIBaseURL)
	if err != nil {
		return "", err
	}
	return text, nil
}

func (o *Orchestrator) SearchAction(queryStr, project string, semantic bool, model string) ([]processor.IndexEntry, error) {
	if semantic {
		return o.runSemanticSearch(queryStr, model, project)
	}
	return o.runBleveSearch(queryStr, project)
}

func (o *Orchestrator) ResearchAction(question, project string, maxDepth int, model string) (string, error) {
	conf, _ := utils.LoadConfig(project)
	if model == "" { model = conf.AIModel }
	
	customPersona := ""
	personaPath := filepath.Join(o.WikiDir, "projects", project, "PROMPT.md")
	if data, err := os.ReadFile(personaPath); err == nil {
		customPersona = string(data)
	}

	client := NewAgentClient(conf.AIBaseURL, model, "", o.WikiDir, project)
	planner := &Planner{Client: client, CustomPersona: customPersona}
	searcher := &Searcher{BaseURL: conf.SearchBaseURL}
	analyst := &Analyst{Client: client, CustomPersona: customPersona}
	proc := processor.NewWikiProcessor(o.WikiDir, project)

	seenURLs := make(map[string]bool)
	currentGoal := question
	var allSources []SourceContent
	var finalReport string

	for depth := 1; depth <= maxDepth; depth++ {
		plan, err := planner.CreatePlan(currentGoal)
		if err != nil { break }

		urls := searcher.DiscoverURLs(plan.Queries, 5)
		var newURLs []string
		for _, u := range urls {
			if !seenURLs[u] {
				seenURLs[u] = true
				newURLs = append(newURLs, u)
			}
		}

		if len(newURLs) > 0 {
			for _, url := range newURLs {
				source := utils.DetectSource(url)
				if source == "" { source = "web" }
				_, text, reliability, _, err := FetchAndSave(source, url, "en", proc, model, conf.AIBaseURL)
				if err == nil {
					allSources = append(allSources, SourceContent{
						ID:          url,
						Content:     text,
						Reliability: reliability,
						SourceType:  source,
					})
				}
			}
		}

		if len(allSources) == 0 { break }

		report, err := analyst.Synthesize(question, allSources)
		if err != nil { break }
		finalReport = report

		recursiveQuery := buildRecursiveQuery(report)
		if recursiveQuery == "" { break }
		currentGoal = recursiveQuery
	}

	proc.UpdateIndex()
	return finalReport, nil
}

func (o *Orchestrator) runBleveSearch(queryStr, project string) ([]processor.IndexEntry, error) {
	blevePath := filepath.Join(o.WikiDir, "musu.bleve")
	if _, err := os.Stat(blevePath); os.IsNotExist(err) { return nil, fmt.Errorf("index not found") }
	index, err := bleve.Open(blevePath)
	if err != nil { return nil, err }
	defer index.Close()

	if project != "all" && project != "" {
		queryStr = fmt.Sprintf("+project:%s %s", project, queryStr)
	}

	query := bleve.NewQueryStringQuery(queryStr)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"title", "source", "id", "summary", "project"}
	searchResults, err := index.Search(searchRequest)
	if err != nil { return nil, err }

	var results []processor.IndexEntry
	for _, hit := range searchResults.Hits {
		results = append(results, processor.IndexEntry{
			ID:      hit.ID,
			Title:   fmt.Sprintf("%v", hit.Fields["title"]),
			Source:  fmt.Sprintf("%v", hit.Fields["source"]),
			Project: fmt.Sprintf("%v", hit.Fields["project"]),
			Summary: fmt.Sprintf("%v", hit.Fields["summary"]),
		})
	}
	return results, nil
}

func (o *Orchestrator) runSemanticSearch(queryStr, model, project string) ([]processor.IndexEntry, error) {
	conf, _ := utils.LoadConfig(project)
	if model == "" { model = conf.AIModel }

	vectorFile := filepath.Join(o.WikiDir, "musu.vectors.json")
	indexFile := filepath.Join(o.WikiDir, "index.json")
	vstore := processor.NewVectorStore()
	if err := vstore.Load(vectorFile); err != nil { return nil, err }

	metaData := make(map[string]processor.IndexEntry)
	if data, err := os.ReadFile(indexFile); err == nil {
		var entries []processor.IndexEntry
		if uerr := json.Unmarshal(data, &entries); uerr != nil {
			fmt.Fprintf(os.Stderr, "warn: parse %s: %v\n", indexFile, uerr)
		}
		for _, e := range entries {
			if project != "all" && project != "" && e.Project != project { continue }
			metaData[e.ID] = e
		}
	}

	client := NewAgentClient(conf.AIBaseURL, model, "", o.WikiDir, project)
	qVec, err := client.Embed(queryStr)
	if err != nil { return nil, err }

	matches := vstore.Search(qVec, 10)
	var results []processor.IndexEntry
	for _, match := range matches {
		if entry, ok := metaData[match.ID]; ok {
			results = append(results, entry)
		}
		if len(results) >= 5 { break }
	}
	return results, nil
}

func buildRecursiveQuery(report string) string {
	recursiveQuery := ""
	contraMatch := regexp.MustCompile(`(?s)CONTRADICTIONS:\s*(.*?)(?:\n\s*MISSING:|$)`).FindStringSubmatch(report)
	if len(contraMatch) >= 2 {
		ci := strings.TrimSpace(contraMatch[1])
		if strings.ToLower(ci) != "none" && len(ci) > 5 {
			recursiveQuery += "Resolve this contradiction: " + ci + ". "
		}
	}
	missingMatch := regexp.MustCompile(`(?s)MISSING:\s*(.*)`).FindStringSubmatch(report)
	if len(missingMatch) >= 2 {
		mi := strings.TrimSpace(missingMatch[1])
		if strings.ToLower(mi) != "none" && len(mi) > 5 {
			recursiveQuery += "Investigate these missing points: " + mi
		}
	}
	return recursiveQuery
}
