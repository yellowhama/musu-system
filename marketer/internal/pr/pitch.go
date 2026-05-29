// Package pr implements the Sprint 4 PR pitch generator for musu-marketer v2.
// It reuses the seo package's grounding (SourcePack) and deterministic
// CitationGate so a media pitch can never hand a journalist a fabricated
// statistic. Pitches are DRAFTS only — they are written to disk for human
// review and are never sent (outbound stays human-in-the-loop, like nurikun).
package pr

import (
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/marketer/internal/seo"
)

// Outlet is a target publication for a pitch. Focus shapes the angle the LLM
// proposes; it is framing only and is never treated as a citable fact.
type Outlet struct {
	Slug  string
	Name  string
	Focus string
}

// DefaultOutlets are Korean agricultural press the NJD story is relevant to.
var DefaultOutlets = []Outlet{
	{Slug: "nongmin", Name: "농민신문", Focus: "농업·농촌 정책과 제도, 농민 권익"},
	{Slug: "aflnews", Name: "한국농어민신문", Focus: "농어업 현장 이슈와 정책 변화"},
	{Slug: "aflnnews", Name: "농수축산신문", Focus: "농수축산 산업 동향과 제도 해설"},
}

// OutletBySlug returns the named outlet, or false if unknown.
func OutletBySlug(slug string) (Outlet, bool) {
	for _, o := range DefaultOutlets {
		if o.Slug == slug {
			return o, true
		}
	}
	return Outlet{}, false
}

// OutletSlugs lists available outlet slugs (CLI help / validation).
func OutletSlugs() []string {
	out := make([]string, len(DefaultOutlets))
	for i, o := range DefaultOutlets {
		out[i] = o.Slug
	}
	return out
}

// Pitch is a finished, gated press pitch ready for human review (NOT sending).
type Pitch struct {
	Outlet  Outlet
	Keyword string
	Subject string // email subject line — framing, not gated
	Angle   string // why this is newsworthy now — framing, not gated
	Body    string // pitch body markdown with [S#] citations — gated
	Pack    seo.SourcePack
}

// Slug is the on-disk filename stem: keyword + outlet.
func (p Pitch) Slug() string {
	return seo.Slugify(p.Keyword + "-" + p.Outlet.Slug)
}

// Render assembles the reviewable pitch document. The header makes the
// human-in-the-loop contract explicit: this is a draft, the operator sends it.
func (p Pitch) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("outlet: %q\n", p.Outlet.Name))
	b.WriteString(fmt.Sprintf("keyword: %q\n", p.Keyword))
	b.WriteString(fmt.Sprintf("subject: %q\n", p.Subject))
	b.WriteString("status: draft\n")
	b.WriteString("delivery: human-in-the-loop (운영자가 직접 검토 후 발송)\n")
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("> ⚠️ 초안입니다. 사실·출처를 검토한 뒤 운영자가 직접 발송하세요. (자동 발송 안 함)\n\n"))
	b.WriteString(fmt.Sprintf("**대상 매체**: %s — %s\n\n", p.Outlet.Name, p.Outlet.Focus))
	b.WriteString(fmt.Sprintf("**제목(안)**: %s\n\n", p.Subject))
	b.WriteString(fmt.Sprintf("**기사 가치(앵글)**: %s\n\n", p.Angle))
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(p.Body))
	b.WriteString("\n\n## 근거 출처\n\n")
	for _, s := range p.Pack.Sources {
		line := fmt.Sprintf("- **[%s]** %s", s.Label, s.Title)
		if s.Source != "" {
			line += fmt.Sprintf(" — %s", s.Source)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
