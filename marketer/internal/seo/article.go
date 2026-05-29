// Package seo implements the Sprint 1 SEO longform blog generator for
// musu-marketer v2. The flow is: ground on the wiki (SourcePack) → LLM outline
// (SEOStrategist) → LLM body (SEOWriter) → deterministic CitationGate → render.
//
// Everything in this file is pure (no LLM, no network) so it is fully unit
// testable; the LLM-driven orchestration lives in generator.go.
package seo

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yellowhama/musu-system/marketer/internal/bridge"
)

// Source is one grounding document with a stable citation label (S1, S2, ...).
// The label is what the LLM must cite and what the CitationGate validates.
type Source struct {
	Label   string // "S1"
	ID      string
	Title   string
	Source  string // provenance, e.g. "법제처" or a URL
	Content string
}

// SourcePack is the ordered set of grounding sources for one article. Labels
// are assigned deterministically (S1..Sn) so a given wiki result always maps to
// the same citation token, which keeps generated articles reproducible.
type SourcePack struct {
	Keyword string
	Sources []Source
}

// NewSourcePack assigns stable [S1..Sn] labels to wiki results in the order the
// bridge returned them (already relevance-sorted).
func NewSourcePack(keyword string, results []bridge.KnowledgeSource) SourcePack {
	pack := SourcePack{Keyword: keyword, Sources: make([]Source, 0, len(results))}
	for i, r := range results {
		pack.Sources = append(pack.Sources, Source{
			Label:   fmt.Sprintf("S%d", i+1),
			ID:      r.ID,
			Title:   r.Title,
			Source:  r.Source,
			Content: r.Content,
		})
	}
	return pack
}

// Labels returns the valid citation labels, e.g. ["S1","S2"].
func (p SourcePack) Labels() []string {
	out := make([]string, len(p.Sources))
	for i, s := range p.Sources {
		out[i] = s.Label
	}
	return out
}

// HasLabel reports whether label is a valid citation in this pack.
func (p SourcePack) HasLabel(label string) bool {
	for _, s := range p.Sources {
		if s.Label == label {
			return true
		}
	}
	return false
}

// Prompt renders the source pack for injection into an LLM prompt: each source
// is fenced with its citation label so the model knows exactly what token to
// emit when it uses that fact.
func (p SourcePack) Prompt() string {
	var b strings.Builder
	for _, s := range p.Sources {
		title := s.Title
		if s.Source != "" {
			title = fmt.Sprintf("%s (출처: %s)", s.Title, s.Source)
		}
		b.WriteString(fmt.Sprintf("### [%s] %s\n%s\n\n", s.Label, title, strings.TrimSpace(s.Content)))
	}
	return b.String()
}

// Outline is the structured SEO plan the SEOStrategist returns as JSON.
type Outline struct {
	Title             string   `json:"title"`
	MetaDescription   string   `json:"meta_description"`
	Slug              string   `json:"slug"`
	TargetKeyword     string   `json:"target_keyword"`
	SecondaryKeywords []string `json:"secondary_keywords"`
	Headings          []string `json:"outline"`
}

// Article is a finished SEO blog post ready to render.
type Article struct {
	Outline Outline
	Body    string // markdown body with [S#] citations
	Pack    SourcePack
}

var slugStrip = regexp.MustCompile(`[^a-z0-9가-힣\-]+`)
var slugDashes = regexp.MustCompile(`\-{2,}`)

// Slugify produces a URL-safe slug. Korean is allowed (네이버/구글 모두 한글
// slug를 인덱싱) but whitespace/punctuation collapse to single dashes.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugStrip.ReplaceAllString(s, "-")
	s = slugDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// jsonLD renders a minimal schema.org Article block. Hand-rolled (no encoding/json)
// so the embedded markdown body is not escaped into the description; description
// is the meta_description only, which we know is plain text.
func (a Article) jsonLD() string {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		s = strings.ReplaceAll(s, "\n", " ")
		return s
	}
	return fmt.Sprintf(`<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "%s",
  "description": "%s",
  "inLanguage": "ko-KR",
  "keywords": "%s"
}
</script>`, esc(a.Outline.Title), esc(a.Outline.MetaDescription),
		esc(strings.Join(append([]string{a.Outline.TargetKeyword}, a.Outline.SecondaryKeywords...), ", ")))
}

// Render assembles the publishable markdown: YAML frontmatter + JSON-LD + body +
// a References section that maps each [S#] back to its title/source.
func (a Article) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", a.Outline.Title))
	b.WriteString(fmt.Sprintf("description: %q\n", a.Outline.MetaDescription))
	b.WriteString(fmt.Sprintf("slug: %q\n", a.Outline.Slug))
	b.WriteString(fmt.Sprintf("target_keyword: %q\n", a.Outline.TargetKeyword))
	if len(a.Outline.SecondaryKeywords) > 0 {
		b.WriteString("secondary_keywords:\n")
		for _, k := range a.Outline.SecondaryKeywords {
			b.WriteString(fmt.Sprintf("  - %q\n", k))
		}
	}
	b.WriteString("lang: ko-KR\n")
	b.WriteString("---\n\n")

	b.WriteString(a.jsonLD())
	b.WriteString("\n\n")

	b.WriteString(strings.TrimSpace(a.Body))
	b.WriteString("\n\n")

	b.WriteString("## 출처\n\n")
	for _, s := range a.Pack.Sources {
		line := fmt.Sprintf("- **[%s]** %s", s.Label, s.Title)
		if s.Source != "" {
			line += fmt.Sprintf(" — %s", s.Source)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
