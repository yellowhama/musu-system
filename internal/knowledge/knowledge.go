package knowledge

// Snippet is a single piece of grounding retrieved for a query.
type Snippet struct {
	Title   string
	Content string
	Source  string  // url / file path / document id
	Score   float64 // relevance, higher is better
}

// Source retrieves grounding snippets for a query. It is configurable per
// deployment so the agent can be dropped into any project:
//   - crawlai.go: queries a musu-crawl-ai wiki (RAG over harvested docs)
//   - folder.go:  embeds and searches a local FAQ/docs folder
//   - none.go:    returns nothing (LLM answers ungrounded)
type Source interface {
	// Retrieve returns up to `limit` snippets relevant to the query.
	Retrieve(query string, limit int) ([]Snippet, error)
	// Name identifies the source for logging/debugging.
	Name() string
}

// NoneSource is the trivial implementation: no grounding. The crawlai and
// folder implementations are added by the knowledge sub-agent.
type NoneSource struct{}

func (NoneSource) Retrieve(string, int) ([]Snippet, error) { return nil, nil }
func (NoneSource) Name() string                            { return "none" }
