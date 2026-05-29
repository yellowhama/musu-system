package seo

import (
	"fmt"
	"strings"
)

// Persona is one NJD (농지다) target audience. A persona changes ONLY the framing,
// tone, examples, and CTA angle of an article — never the facts. The same grounded
// SourcePack and the same deterministic CitationGate apply to every persona variant,
// so persona retelling can never introduce an uncited legal claim. This is what keeps
// "audience reach multiplier" on the healthy side of the line: same verified law,
// many honest framings.
type Persona struct {
	Slug      string // url/filename suffix, e.g. "elderly-owner"
	Name      string // "고령 농지주"
	Situation string // who they are, one line
	Pain      string // their core worry — what they came to solve
	Voice     string // tone + reading-level guidance for the LLM
	CTAAngle  string // which honest NJD value resonates (never fear-based)
}

// DefaultPersonas is the NJD audience set. Four distinct intents that all map back
// to the same farmland-law knowledge base but read very differently.
var DefaultPersonas = []Persona{
	{
		Slug:      "elderly-owner",
		Name:      "고령 농지주",
		Situation: "오래 농지를 보유했지만 직접 농사를 못 짓게 된 60~80대 소유자",
		Pain:      "처분명령·이행강제금 같은 말이 무섭고 어디에 물어야 할지 모름",
		Voice:     "아주 쉽고 차분하게. 전문용어는 풀어서 한 번 더 설명. 짧은 문장. 어르신을 안심시키되 과장 없이.",
		CTAAngle:  "복잡한 절차를 대신 차근차근 안내받을 수 있다는 안도감",
	},
	{
		Slug:      "heir",
		Name:      "농지 상속인",
		Situation: "부모에게서 농지를 상속받았지만 본인은 농사를 짓지 않는 도시 거주자",
		Pain:      "상속 농지를 계속 가질 수 있는지, 언제까지 무엇을 해야 하는지 헷갈림",
		Voice:     "실무적이고 명확하게. 기한과 선택지를 구조적으로. 바쁜 직장인이 빠르게 핵심을 잡도록.",
		CTAAngle:  "내 상황에 맞는 선택지(보유·처분·임대)를 빠르게 정리받는 효율",
	},
	{
		Slug:      "returning-farmer",
		Name:      "귀농 준비자",
		Situation: "도시를 떠나 농촌 정착·농지 매입을 준비 중인 30~50대",
		Pain:      "농지를 합법적으로 잘 사서 문제없이 시작하고 싶음",
		Voice:     "차근하고 긍정적으로. 준비 체크리스트 느낌. 처음 접하는 제도를 단계로 안내.",
		CTAAngle:  "첫 농지 취득을 실수 없이 밟도록 같이 점검받는다는 신뢰",
	},
	{
		Slug:      "prospective-buyer",
		Name:      "농지 관심 일반인",
		Situation: "농지 취득·활용에 관심은 있으나 법적 제약이 걱정인 신중한 일반인",
		Pain:      "농지는 아무나 못 산다던데 내 경우 가능한지, 규제가 정확히 뭔지 알고 싶음",
		Voice:     "사실 중심, 담백하게. 투기·한탕 뉘앙스 절대 금지. 가능/불가능 조건을 정확히.",
		CTAAngle:  "정확한 사실에 근거해 가능 여부를 따져볼 수 있다는 명료함",
	},
}

// PersonaBySlug returns the named persona, or false if unknown.
func PersonaBySlug(slug string) (Persona, bool) {
	for _, p := range DefaultPersonas {
		if p.Slug == slug {
			return p, true
		}
	}
	return Persona{}, false
}

// PersonaSlugs lists the available persona slugs (for CLI help / validation).
func PersonaSlugs() []string {
	out := make([]string, len(DefaultPersonas))
	for i, p := range DefaultPersonas {
		out[i] = p.Slug
	}
	return out
}

// promptBlock renders the persona as an LLM instruction block. Returns "" for the
// zero Persona so the generic (persona-less) path injects nothing.
func (p Persona) promptBlock() string {
	if p.Name == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("### 독자 페르소나 (프레이밍 전용 — 사실은 절대 바꾸지 말 것) ###\n")
	b.WriteString(fmt.Sprintf("- 대상: %s — %s\n", p.Name, p.Situation))
	b.WriteString(fmt.Sprintf("- 핵심 고민: %s\n", p.Pain))
	b.WriteString(fmt.Sprintf("- 어조/난이도: %s\n", p.Voice))
	b.WriteString(fmt.Sprintf("- CTA 방향(공포 금지): %s\n", p.CTAAngle))
	b.WriteString("이 페르소나에 맞게 제목·구성·예시·말투만 조정하고, 법률/수치/기관 사실과 출처 라벨은 동일하게 유지하세요.\n")
	return b.String()
}
