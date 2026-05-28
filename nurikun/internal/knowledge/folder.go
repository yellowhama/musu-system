package knowledge

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// maxExcerptLen caps the length of the excerpt returned in a Snippet.
const maxExcerptLen = 600

// folderDoc is one local document loaded into memory.
type folderDoc struct {
	title string
	path  string
	text  string
}

// folderSource searches a local folder of .md/.txt documents using a simple,
// dependency-free keyword-overlap score. No embeddings, no network.
type folderSource struct {
	docs []folderDoc
}

// newFolderSource walks s.Dir for *.md and *.txt files and loads each into
// memory. It returns an error if the directory is unset or unreadable.
func newFolderSource(s Settings) (*folderSource, error) {
	if s.Dir == "" {
		return nil, fmt.Errorf("folder: Dir is required")
	}
	info, err := os.Stat(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("folder: stat %q: %w", s.Dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("folder: %q is not a directory", s.Dir)
	}

	var docs []folderDoc
	walkErr := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("folder: read %q: %w", path, readErr)
		}
		text := string(raw)
		docs = append(docs, folderDoc{
			title: docTitle(text, path),
			path:  path,
			text:  text,
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("folder: walk %q: %w", s.Dir, walkErr)
	}

	return &folderSource{docs: docs}, nil
}

func (f *folderSource) Name() string { return "folder" }

// Retrieve scores every document by case-insensitive query-term overlap and
// returns the top `limit` as Snippets. The Content is the paragraph with the
// most query-term hits, capped at maxExcerptLen.
func (f *folderSource) Retrieve(query string, limit int) ([]Snippet, error) {
	if limit <= 0 {
		return nil, nil
	}
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	type scored struct {
		doc   folderDoc
		score float64
	}
	var hits []scored
	for _, doc := range f.docs {
		score := overlapScore(doc.text, terms)
		if score > 0 {
			hits = append(hits, scored{doc: doc, score: score})
		}
	}

	// Highest score first; stable tiebreak on path for determinism.
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].doc.path < hits[j].doc.path
	})

	if len(hits) > limit {
		hits = hits[:limit]
	}

	snippets := make([]Snippet, 0, len(hits))
	for _, h := range hits {
		snippets = append(snippets, Snippet{
			Title:   h.doc.title,
			Content: bestExcerpt(h.doc.text, terms),
			Source:  h.doc.path,
			Score:   h.score,
		})
	}
	return snippets, nil
}

// overlapScore is the total number of query-term occurrences in the text
// (term-frequency overlap), matched case-insensitively on whole tokens.
func overlapScore(text string, terms []string) float64 {
	counts := termCounts(tokenize(text))
	var score float64
	for _, t := range terms {
		score += float64(counts[t])
	}
	return score
}

// bestExcerpt returns the paragraph containing the most query-term hits,
// trimmed and capped at maxExcerptLen runes.
func bestExcerpt(text string, terms []string) string {
	paragraphs := splitParagraphs(text)
	best := ""
	bestScore := -1.0
	for _, p := range paragraphs {
		s := overlapScore(p, terms)
		if s > bestScore {
			bestScore = s
			best = p
		}
	}
	if best == "" {
		best = strings.TrimSpace(text)
	}
	best = strings.TrimSpace(best)
	return cap600(best)
}

func cap600(s string) string {
	r := []rune(s)
	if len(r) <= maxExcerptLen {
		return s
	}
	return strings.TrimSpace(string(r[:maxExcerptLen]))
}

// splitParagraphs breaks text on blank lines.
func splitParagraphs(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(normalized, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// docTitle uses the first Markdown heading (# ...) if present, otherwise the
// filename without extension.
func docTitle(text, path string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if heading != "" {
				return heading
			}
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// tokenize lowercases and splits text into alphanumeric word tokens.
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	return fields
}

func termCounts(tokens []string) map[string]int {
	counts := make(map[string]int, len(tokens))
	for _, t := range tokens {
		counts[t]++
	}
	return counts
}
