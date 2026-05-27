package knowledge

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// crawlAISource retrieves grounding by shelling out to the musu-crawl-ai
// binary, which performs RAG over a project's harvested wiki/docs.
type crawlAISource struct {
	path    string // path to the musu-crawl-ai executable
	project string // crawl-ai project to search
}

// newCrawlAISource builds a crawlAISource. It requires a non-empty CrawlPath
// so it can locate the musu-crawl-ai binary.
func newCrawlAISource(s Settings) (*crawlAISource, error) {
	if s.CrawlPath == "" {
		return nil, fmt.Errorf("crawlai: CrawlPath is required")
	}
	return &crawlAISource{path: s.CrawlPath, project: s.Project}, nil
}

func (c *crawlAISource) Name() string { return "crawlai" }

// crawlEnvelope is the JSON shape emitted by `musu-crawl-ai search ... --json`.
//
//	{
//	  "status": "success",
//	  "message": "...",
//	  "data": [
//	    {"id":"..","title":"..","source":"..","project":"..","summary":".."},
//	    ...
//	  ]
//	}
type crawlEnvelope struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    []crawlResult `json:"data"`
}

type crawlResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	Project string `json:"project"`
	Summary string `json:"summary"`
}

// Retrieve runs the crawl-ai search subcommand and maps its results to
// Snippets. It degrades gracefully: a non-success status or missing data
// yields an empty result rather than an error, so the agent can still answer
// (ungrounded) when the crawler is unavailable or returns nothing.
func (c *crawlAISource) Retrieve(query string, limit int) ([]Snippet, error) {
	if limit <= 0 {
		return nil, nil
	}

	cmd := exec.Command(c.path, "search", query, "--project", c.project, "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("crawlai: search command failed: %w", err)
	}

	var env crawlEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("crawlai: parse search output: %w", err)
	}

	// Degrade gracefully: nothing usable to ground on.
	if env.Status != "success" || len(env.Data) == 0 {
		return nil, nil
	}

	results := env.Data
	if len(results) > limit {
		results = results[:limit]
	}

	snippets := make([]Snippet, 0, len(results))
	n := len(results)
	for i, r := range results {
		snippets = append(snippets, Snippet{
			Title:   r.Title,
			Content: r.Summary,
			Source:  r.ID,
			// Descending by rank: first result scores highest.
			Score: float64(n - i),
		})
	}
	return snippets, nil
}
