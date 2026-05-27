package knowledge

// Settings carries knowledge-source configuration. The caller populates it
// from config.
type Settings struct {
	Kind      string // "crawlai" | "folder" | "none"
	CrawlPath string // path to musu-crawl.exe (crawlai)
	Project   string // crawl-ai project to search (crawlai)
	Dir       string // local docs directory (folder)
	AIBaseURL string // OpenAI-compatible base URL for embeddings (folder)
	AIModel   string
}

// New constructs a knowledge Source. Defaults to NoneSource (no grounding).
//
// The knowledge sub-agent replaces newCrawlAI/newFolder with real
// implementations (crawlai.go / folder.go). Until then they fall back to
// NoneSource so the rest of the tree compiles against this factory.
func New(s Settings) (Source, error) {
	switch s.Kind {
	case "crawlai":
		return newCrawlAI(s)
	case "folder":
		return newFolder(s)
	default: // "none" or empty
		return NoneSource{}, nil
	}
}

func newCrawlAI(Settings) (Source, error) { return NoneSource{}, nil }
func newFolder(Settings) (Source, error)  { return NoneSource{}, nil }
