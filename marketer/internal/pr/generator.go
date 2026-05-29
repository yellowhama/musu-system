package pr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/marketer/internal/agent"
	"github.com/yellowhama/musu-system/marketer/internal/bridge"
	"github.com/yellowhama/musu-system/marketer/internal/seo"
)

// Generator produces gated press pitches. It mirrors seo.Generator's wiring
// (shared agent client + wiki bridge) but reuses seo.SourcePack / CheckCitations
// so the "no fabricated stat to a journalist" guarantee is the same gate.
type Generator struct {
	Client      *agent.AgentClient
	Wiki        *bridge.WikiBridge
	MaxRewrites int
}

func NewPitchGenerator(aiURL, model, wikiDir, project string) *Generator {
	return &Generator{
		Client:      agent.NewAgentClient(aiURL, model, wikiDir, project),
		Wiki:        bridge.NewWikiBridge(wikiDir),
		MaxRewrites: 2,
	}
}

type pitchDraft struct {
	Subject string `json:"subject"`
	Angle   string `json:"angle"`
	Body    string `json:"body"`
}

// Generate writes a gated pitch for one keyword + outlet. When strict and the
// body still carries uncited claims after MaxRewrites, it returns an error and
// the pitch MUST NOT be sent — same hard gate as the SEO path.
func (g *Generator) Generate(keyword string, outlet Outlet, strict bool) (*Pitch, seo.CitationReport, error) {
	results, err := g.Wiki.FindByTopic(keyword)
	if err != nil || len(results) == 0 {
		return nil, seo.CitationReport{}, fmt.Errorf("no verified knowledge found for keyword: %s", keyword)
	}
	pack := seo.NewSourcePack(keyword, results)

	var (
		draft    pitchDraft
		rep      seo.CitationReport
		feedback string
	)
	for attempt := 0; attempt <= g.MaxRewrites; attempt++ {
		draft, err = g.draft(keyword, outlet, pack, feedback)
		if err != nil {
			return nil, seo.CitationReport{}, fmt.Errorf("pr draft phase failed: %w", err)
		}
		rep = seo.CheckCitations(draft.Body, pack)
		if rep.OK(strict) {
			break
		}
		feedback = citationFeedback(rep)
	}

	pitch := &Pitch{
		Outlet:  outlet,
		Keyword: keyword,
		Subject: strings.TrimSpace(draft.Subject),
		Angle:   strings.TrimSpace(draft.Angle),
		Body:    draft.Body,
		Pack:    pack,
	}
	if gateErr := rep.Err(strict); gateErr != nil {
		return pitch, rep, gateErr
	}
	return pitch, rep, nil
}

func (g *Generator) draft(keyword string, outlet Outlet, pack seo.SourcePack, feedback string) (pitchDraft, error) {
	fb := ""
	if feedback != "" {
		fb = fmt.Sprintf("\n### 이전 시도 반려 — 반드시 수정 ###\n%s\n", feedback)
	}
	prompt := fmt.Sprintf(`당신은 농지·농지법 전문 홍보 담당자입니다. 아래 검증된 출처만 근거로,
"%s" 주제에 대해 매체 "%s"(관심사: %s)에 보낼 보도자료 피치 초안을 작성하세요.

### 검증된 출처 (이 안의 사실만 사용) ###
%s
%s
### 절대 규칙 ###
1. body의 법률·수치·날짜·기관 주장 문장은 끝에 출처 라벨을 붙인다. 예: "처분명령은 농지법 제10조에 근거한다 [S1]."
2. 출처에 없는 통계/사실은 절대 지어내지 않는다. 기자에게 가는 글이므로 더욱 엄격히.
3. 라벨은 [S1]..[S%d] 중 실제 존재하는 것만 사용한다.
4. 공포·과장·홍보성 미사여구 금지. 공익적 뉴스 가치 중심.
5. subject는 한 줄 제목, angle은 "왜 지금 기사가치가 있는지" 2~3문장, body는 기자가 바로 읽을 수 있는 본문.

엄격한 JSON만 출력:
{"subject": "...", "angle": "...", "body": "마크다운 본문 [S#] 인용 포함"}`,
		keyword, outlet.Name, outlet.Focus, pack.Prompt(), fb, len(pack.Sources))

	resp, err := g.Client.Ask(prompt, true)
	if err != nil {
		return pitchDraft{}, err
	}
	var d pitchDraft
	if err := json.Unmarshal([]byte(resp), &d); err != nil {
		return pitchDraft{}, fmt.Errorf("pitch JSON parse: %w", err)
	}
	if strings.TrimSpace(d.Body) == "" {
		return pitchDraft{}, fmt.Errorf("pitch body empty")
	}
	return d, nil
}

// citationFeedback mirrors the SEO loop's targeted rewrite instructions.
func citationFeedback(rep seo.CitationReport) string {
	var b strings.Builder
	if len(rep.InvalidLabels) > 0 {
		b.WriteString(fmt.Sprintf("- 존재하지 않는 출처 라벨 사용: %s. 실제 출처 라벨만 쓰세요.\n",
			strings.Join(rep.InvalidLabels, ", ")))
	}
	if len(rep.UncitedClaims) > 0 {
		b.WriteString("- 다음 주장에 출처가 없습니다. 출처를 붙이거나 삭제하세요:\n")
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
