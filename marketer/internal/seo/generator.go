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
	Polish      bool // run the magazine-class editorial rewrite pass
}

// NewGenerator wires a generator against the shared Ollama-backed agent client.
func NewGenerator(aiURL, model, wikiDir, project string, minWords int) *Generator {
	return &Generator{
		Client:      agent.NewAgentClient(aiURL, model, wikiDir, project),
		Wiki:        bridge.NewWikiBridge(wikiDir),
		MinWords:    minWords,
		MaxRewrites: 3,
		Polish:      true,
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
	if outline.TargetKeyword == "" {
		outline.TargetKeyword = keyword
	}
	// Derive a clean, deterministic slug from the target keyword rather than
	// trusting the model's often-garbled romanization. Persona disambiguates.
	slugBase := outline.TargetKeyword
	if persona.Slug != "" {
		slugBase = slugBase + "-" + persona.Slug
	}
	outline.Slug = Slugify(slugBase)

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

	if gateErr := rep.Err(strict); gateErr != nil {
		return &Article{Outline: *outline, Body: body, Pack: pack}, rep, gateErr
	}

	// Editorial polish: rewrite the gated draft to magazine-class prose while
	// preserving every citation and fact. Re-gate the result; only keep the
	// polish if it still passes AND retains the cited-claim count (a polish that
	// drops or weakens citations is rejected in favour of the accurate draft).
	if g.Polish {
		if polished, perr := g.polish(body, pack, persona); perr == nil {
			prep := CheckCitations(polished, pack)
			if prep.OK(strict) && prep.CitedClaims >= rep.CitedClaims {
				body, rep = polished, prep
			}
		}
	}

	art := &Article{Outline: *outline, Body: body, Pack: pack}
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

### 문체 — 잡지 기사 클래스 (반드시) ###
- 도입부는 구체적 장면이나 독자의 실제 상황으로 시작한다. "농지 소유자라면 …할 수 있습니다" 같은 교과서식 도입 금지.
- AI 상투어·군더더기 금지: "중요한 역할을 합니다", "~에 대해 알아보겠습니다", "결론적으로", "~라는 점에서 의미가 있습니다" 류 삭제.
- 문장 길이를 변주한다. 짧은 단정문과 설명문을 섞어 리듬을 만든다.
- 전문가가 옆에서 차분히 설명하듯, 따뜻하지만 군더더기 없는 목소리.
- 같은 사실을 반복 부연하지 말 것. 한 번 명확히 말하고 넘어간다.

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

// polish runs an editorial rewrite to lift the gated draft to magazine-class
// prose. It is hard-constrained to preserve every [S#] citation and invent no
// new fact; the caller re-gates the result and discards it if any citation was
// dropped or weakened. This keeps editorial quality from ever costing accuracy.
func (g *Generator) polish(body string, pack SourcePack, persona Persona) (string, error) {
	voice := "전문적이면서 따뜻하고 신뢰감 있는"
	if persona.Voice != "" {
		voice = persona.Voice
	}
	prompt := fmt.Sprintf(`당신은 한국 유력 시사·실용 매거진의 시니어 에디터입니다.
아래 초안을 '잡지 기사 클래스'로 다시 씁니다. 정보는 정확하나 문장이 평범하고 템플릿틱한 초안입니다.

### 다시 쓰기 원칙 ###
- 강렬한 도입부(리드): 독자의 실제 상황·장면·핵심 질문으로 시작. 교과서식 도입 제거.
- 내러티브 흐름과 문장 리듬: 짧은 문장과 긴 문장을 섞고, 단락을 자연스럽게 잇는다.
- AI 상투어·군더더기·동어반복 제거. 한 번 말한 사실을 부연 반복하지 않는다.
- 목소리: %s 톤. 전문가가 곁에서 설명하듯.
- 소제목(H2/H3)도 정보적이되 읽고 싶게 다듬는다.

### 절대 불변 (어기면 실패) ###
1. 모든 [S#] 출처 라벨을 해당 사실 문장에 **그대로 유지**한다. 라벨을 옮기되 사실과 분리하지 말 것.
2. 초안에 없는 사실·수치·법조문·기관을 **새로 추가하지 않는다**. 인용 없는 일반 진술(예: "벌금을 물 수 있다")도 새로 만들지 말 것.
3. 마크다운 본문만 출력. frontmatter·메타·설명 라벨 금지.

### 초안 ###
%s

### 출처 라벨 목록 (이 라벨만 존재) ###
%s

다시 쓴 본문(마크다운)만 출력:`, voice, body, strings.Join(pack.Labels(), ", "))

	resp, err := g.Client.Ask(prompt, false)
	if err != nil {
		return "", err
	}
	resp = sanitizeBody(resp)
	if strings.TrimSpace(resp) == "" {
		return "", fmt.Errorf("polish returned empty body")
	}
	return resp, nil
}

// metaEchoLine matches lines where the model echoed outline/frontmatter fields
// (e.g. "**meta_description:** ...", "target_keyword: ...") into the body. These
// are generation artifacts, never legitimate prose, and would otherwise trip the
// citation gate as uncited claims.
var metaEchoLine = regexp.MustCompile(`(?i)^\s*[*_#>\s]*(meta[_\s]?description|target[_\s]?keyword|secondary[_\s]?keywords|slug|title|outline|description|메타\s*설명|메타\s*디스크립션|디스크립션|메타데이터|타[게깃]\s*키워드|핵심\s*키워드|보조\s*키워드|키워드|제목|슬러그|더보기|요약)\s*[*_]*\s*[:：]`)

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
