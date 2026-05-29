package seo

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/yellowhama/musu-system/marketer/internal/agent"
	"github.com/yellowhama/musu-system/marketer/internal/bridge"
)

// Generator orchestrates the LLM phases of SEO blog generation. The deterministic
// pieces (SourcePack, CitationGate, Render) live alongside in this package and are
// exercised directly by unit tests; Generator is the integration seam.
type Generator struct {
	Client      *agent.AgentClient
	Wiki        *bridge.WikiBridge
	MinWords    int
	MaxRewrites int
}

// NewGenerator wires a generator against the shared Ollama-backed agent client.
func NewGenerator(aiURL, model, wikiDir, project string, minWords int) *Generator {
	return &Generator{
		Client:      agent.NewAgentClient(aiURL, model, wikiDir, project),
		Wiki:        bridge.NewWikiBridge(wikiDir),
		MinWords:    minWords,
		MaxRewrites: 3,
	}
}

// Generate runs the full SEO pipeline for one keyword with no persona framing
// (the generic article). See generate for the shared implementation.
func (g *Generator) Generate(keyword string, strict bool) (*Article, CitationReport, error) {
	return g.generate(keyword, Persona{}, strict)
}

// GenerateForPersona runs the pipeline retold for a specific NJD audience. The
// facts, sources, and citation gate are identical to Generate; only framing,
// tone, examples, and CTA angle differ. The persona slug is appended to the
// article slug so variants of the same keyword never collide on disk.
func (g *Generator) GenerateForPersona(keyword string, persona Persona, strict bool) (*Article, CitationReport, error) {
	return g.generate(keyword, persona, strict)
}

// VariantResult pairs a persona with its generated article (or the error/report
// from a failed gate) so GenerateVariants can return partial success.
type VariantResult struct {
	Persona Persona
	Article *Article
	Report  CitationReport
	Err     error
}

// GenerateVariants fans a single keyword out across personas. Each variant is
// generated and gated independently; one persona failing the citation gate does
// not abort the others. Callers publish only the variants with Err == nil.
func (g *Generator) GenerateVariants(keyword string, personas []Persona, strict bool) []VariantResult {
	out := make([]VariantResult, 0, len(personas))
	for _, p := range personas {
		art, rep, err := g.generate(keyword, p, strict)
		out = append(out, VariantResult{Persona: p, Article: art, Report: rep, Err: err})
	}
	return out
}

// generate is the shared pipeline. A zero Persona means the generic article.
// When strict is true and the body still carries uncited claims after
// MaxRewrites, it returns an error and the caller MUST NOT publish — this is the
// "no unsanitary food" hard gate.
func (g *Generator) generate(keyword string, persona Persona, strict bool) (*Article, CitationReport, error) {
	results, err := g.Wiki.FindByTopic(keyword)
	if err != nil || len(results) == 0 {
		return nil, CitationReport{}, fmt.Errorf("no verified knowledge found for keyword: %s", keyword)
	}
	pack := NewSourcePack(keyword, results)

	outline, err := g.outline(keyword, pack, persona)
	if err != nil {
		return nil, CitationReport{}, fmt.Errorf("seo outline phase failed: %w", err)
	}
	if strings.TrimSpace(outline.Slug) == "" {
		outline.Slug = Slugify(outline.Title)
	}
	if outline.TargetKeyword == "" {
		outline.TargetKeyword = keyword
	}
	// Disambiguate per-persona files for the same keyword.
	if persona.Slug != "" {
		outline.Slug = Slugify(outline.Slug + "-" + persona.Slug)
	}

	var (
		body     string
		rep      CitationReport
		feedback string
	)
	// Citation self-correction loop: regenerate while uncited claims remain, up
	// to MaxRewrites. The feedback handed back to the writer is the concrete list
	// of sentences that lacked a source, so the rewrite is targeted.
	for attempt := 0; attempt <= g.MaxRewrites; attempt++ {
		body, err = g.write(keyword, outline, pack, feedback, persona)
		if err != nil {
			return nil, CitationReport{}, fmt.Errorf("seo write phase failed: %w", err)
		}
		rep = CheckCitations(body, pack)
		if rep.OK(strict) {
			break
		}
		feedback = citationFeedback(rep)
	}

	art := &Article{Outline: *outline, Body: body, Pack: pack}
	if gateErr := rep.Err(strict); gateErr != nil {
		return art, rep, gateErr
	}
	return art, rep, nil
}

func (g *Generator) outline(keyword string, pack SourcePack, persona Persona) (*Outline, error) {
	prompt := fmt.Sprintf(`당신은 한국어 SEO 콘텐츠 전략가입니다. 아래 검증된 출처만 근거로,
키워드 "%s"에 대한 블로그 글의 SEO 구조를 설계하세요.

### 검증된 출처 ###
%s
%s
### 작업 ###
1. 검색 의도에 맞는 제목(title) — 키워드 포함, 60자 이내
2. meta_description — 160자 이내, 클릭 유도, 과장/공포 금지
3. slug — URL용 (영문 또는 한글, 공백은 하이픈)
4. target_keyword — 핵심 키워드 1개
5. secondary_keywords — 보조 키워드 3~5개
6. outline — H2/H3 제목 5~8개 (각 항목이 출처로 뒷받침 가능해야 함)

엄격한 JSON만 출력:
{
  "title": "...",
  "meta_description": "...",
  "slug": "...",
  "target_keyword": "...",
  "secondary_keywords": ["...", "..."],
  "outline": ["H2: ...", "  H3: ...", "H2: ..."]
}`, keyword, pack.Prompt(), persona.promptBlock())

	resp, err := g.Client.Ask(prompt, true)
	if err != nil {
		return nil, err
	}
	var o Outline
	if err := json.Unmarshal([]byte(resp), &o); err != nil {
		return nil, fmt.Errorf("outline JSON parse: %w", err)
	}
	return &o, nil
}

func (g *Generator) write(keyword string, outline *Outline, pack SourcePack, feedback string, persona Persona) (string, error) {
	fb := ""
	if feedback != "" {
		fb = fmt.Sprintf("\n### 이전 시도 반려 — 반드시 수정 ###\n%s\n", feedback)
	}
	outlineJSON, _ := json.MarshalIndent(outline, "", "  ")

	prompt := fmt.Sprintf(`당신은 농지·농지법 전문 한국어 블로그 작가입니다.
아래 구조와 검증된 출처만 사용해 본문(마크다운)을 작성하세요.

### SEO 구조 ###
%s

### 검증된 출처 (이 안의 사실만 사용) ###
%s
%s%s
### 절대 규칙 ###
1. 법률·수치·날짜·기관에 대한 모든 주장 문장은 끝에 해당 출처 라벨을 붙인다. 예: "2026년 농지 전수조사가 시행된다 [S2]."
2. **한 출처에서 여러 문장을 연속으로 쓸 때는, 각 문장 끝에 같은 라벨을 반복한다.** 앞 문장에 라벨을 붙였다고 다음 문장을 생략하지 않는다. 예: "처분의무가 생긴 소유자는 1년 이내에 처분해야 한다 [S1]. 처분하지 않으면 6개월 이내 처분명령이 내려질 수 있다 [S1]."
3. 출처에 없는 사실/법조문은 절대 지어내지 않는다. 모르면 쓰지 않는다. 출처로 뒷받침되지 않는 일반론(예: "효율성을 높인다")도 사실 주장처럼 쓰지 말 것.
4. 라벨은 반드시 위 출처에 존재하는 것만 사용한다 ([S1]..[S%d]).
5. 공포 마케팅·압박("지금 안 하면 처분!") 금지. 정보 가치 중심.
6. 최소 %d단어 분량, H2/H3 마크다운 사용.

본문 마크다운만 출력 (frontmatter 없이):`,
		string(outlineJSON), pack.Prompt(), fb, persona.promptBlock(), len(pack.Sources), g.MinWords)

	resp, err := g.Client.Ask(prompt, false)
	if err != nil {
		return "", err
	}
	resp = sanitizeBody(resp)
	if strings.TrimSpace(resp) == "" {
		return "", fmt.Errorf("writer returned empty body")
	}
	return resp, nil
}

// metaEchoLine matches lines where the model echoed outline/frontmatter fields
// (e.g. "**meta_description:** ...", "target_keyword: ...") into the body. These
// are generation artifacts, never legitimate prose, and would otherwise trip the
// citation gate as uncited claims.
var metaEchoLine = regexp.MustCompile(`(?i)^\s*[*_#>\s]*(meta[_\s]?description|target[_\s]?keyword|secondary[_\s]?keywords|slug|title|outline|description|메타\s*설명|메타데이터|타[게깃]\s*키워드|핵심\s*키워드|보조\s*키워드|키워드|제목|슬러그)\s*[*_]*\s*[:：]`)

// frontmatterFence strips a leading YAML/JSON frontmatter block if the model
// emitted one despite being told not to.
var frontmatterFence = regexp.MustCompile("(?s)^\\s*(---.*?---|```[a-z]*\\s*\\{.*?\\}\\s*```)\\s*")

func sanitizeBody(body string) string {
	body = frontmatterFence.ReplaceAllString(body, "")
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	for _, ln := range lines {
		if metaEchoLine.MatchString(ln) {
			continue
		}
		kept = append(kept, ln)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// citationFeedback turns a failing report into targeted rewrite instructions.
func citationFeedback(rep CitationReport) string {
	var b strings.Builder
	if len(rep.InvalidLabels) > 0 {
		b.WriteString(fmt.Sprintf("- 존재하지 않는 출처 라벨을 사용함: %s. 위 출처 목록에 있는 라벨만 사용하세요.\n",
			strings.Join(rep.InvalidLabels, ", ")))
	}
	if len(rep.UncitedClaims) > 0 {
		b.WriteString("- 다음 주장 문장들에 출처 [S#]가 없습니다. 출처를 붙이거나, 출처로 뒷받침 못 하면 삭제하세요:\n")
		for i, c := range rep.UncitedClaims {
			if i >= 5 {
				b.WriteString(fmt.Sprintf("  ... 외 %d건\n", len(rep.UncitedClaims)-5))
				break
			}
			b.WriteString(fmt.Sprintf("  · %s\n", c))
		}
	}
	return b.String()
}
