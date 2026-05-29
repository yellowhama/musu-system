// Package funnel turns an acquisition target (e.g. 10,000 NJD members) into a
// transparent, deterministic campaign plan over the tools musu-marketer already
// has: SEO articles, persona variants, PR pitches, opt-in capture, and
// human-in-the-loop nurture. Everything here is pure (no LLM, no network) so the
// plan and its arithmetic are fully reproducible and unit-testable. The point is
// to make "10,000 members" concrete: it shows exactly how much healthy traffic
// and opt-in performance the target requires, and a coordinated calendar to get
// there — the kind of strategy deliverable a commercial agency hands a client.
package funnel

import (
	"fmt"
	"math"
	"strings"
)

// Stage is a classic acquisition funnel stage.
type Stage string

const (
	TOFU Stage = "TOFU" // top: organic discovery (SEO, PR)
	MOFU Stage = "MOFU" // middle: opt-in capture + nurture
	BOFU Stage = "BOFU" // bottom: conversion to member / consult
)

// Action is one scheduled step. Day is a relative offset (Day 1, Day 3, ...) so
// the plan is deterministic and stable regardless of when it is generated.
type Action struct {
	Day     int
	Stage   Stage
	Tool    string // "seo", "seo --all-personas", "pr --all-outlets", "opt-in", "nurture", "referral"
	Command string // concrete command or manual step
	Goal    string
}

// Config holds the (operator-tunable) acquisition assumptions. Defaults are
// deliberately conservative and MUST be validated against real analytics — the
// plan prints them as assumptions, never as promises.
type Config struct {
	TargetMembers      int     // e.g. 10000
	OptInRate          float64 // visitor -> opt-in subscriber (healthy lead magnet), e.g. 0.03
	MemberRate         float64 // opt-in -> activated member, e.g. 0.5
	VisitsPerArticleMo int     // sustained monthly organic visits per indexed article
	PersonaMultiplier  float64 // extra reach from persona variants of one keyword
	HorizonMonths      int     // campaign horizon for the "articles needed" estimate
}

// DefaultConfig returns conservative defaults toward a 10,000-member target.
func DefaultConfig() Config {
	return Config{
		TargetMembers:      10000,
		OptInRate:          0.03,
		MemberRate:         0.5,
		VisitsPerArticleMo: 150,
		PersonaMultiplier:  1.6,
		HorizonMonths:      12,
	}
}

// Math is the derived, transparent arithmetic from a Config.
type Math struct {
	VisitorsNeeded int // total visitors to hit the member target
	OptInsNeeded   int // opt-in subscribers required
	MonthlyVisits  int // visits/month required over the horizon
	ArticlesNeeded int // SEO articles (with persona reach) to sustain that traffic
}

// Compute derives the funnel arithmetic. Guards against zero rates.
func (c Config) Compute() Math {
	optIn := c.OptInRate
	mem := c.MemberRate
	if optIn <= 0 {
		optIn = 0.03
	}
	if mem <= 0 {
		mem = 0.5
	}
	optInsNeeded := math.Ceil(float64(c.TargetMembers) / mem)
	visitorsNeeded := math.Ceil(optInsNeeded / optIn)

	horizon := c.HorizonMonths
	if horizon <= 0 {
		horizon = 12
	}
	monthlyVisits := math.Ceil(visitorsNeeded / float64(horizon))

	perArticle := float64(c.VisitsPerArticleMo) * math.Max(1, c.PersonaMultiplier)
	if perArticle <= 0 {
		perArticle = 1
	}
	articles := math.Ceil(monthlyVisits / perArticle)

	return Math{
		VisitorsNeeded: int(visitorsNeeded),
		OptInsNeeded:   int(optInsNeeded),
		MonthlyVisits:  int(monthlyVisits),
		ArticlesNeeded: int(articles),
	}
}

// Plan is a coordinated campaign across keywords plus standing funnel actions.
type Plan struct {
	Project  string
	Keywords []string
	Config   Config
	Math     Math
	Actions  []Action
}

// BuildPlan schedules a deterministic campaign. Per keyword: a pillar SEO
// article, persona variants two days later, and a PR pitch a few days after.
// Standing MOFU/BOFU actions (opt-in, nurture, referral) anchor the funnel.
func BuildPlan(project string, keywords []string, cfg Config) Plan {
	p := Plan{Project: project, Keywords: keywords, Config: cfg, Math: cfg.Compute()}

	day := 1
	for _, kw := range keywords {
		p.Actions = append(p.Actions,
			Action{Day: day, Stage: TOFU, Tool: "seo",
				Command: fmt.Sprintf("musu-marketer seo %q -p %s", kw, project),
				Goal:    "검색 의도 충족 pillar 글 (출처 게이트 통과)"},
			Action{Day: day + 2, Stage: TOFU, Tool: "seo --all-personas",
				Command: fmt.Sprintf("musu-marketer seo %q -p %s --all-personas", kw, project),
				Goal:    "동일 사실, 4개 페르소나 프레이밍으로 도달 확장"},
			Action{Day: day + 5, Stage: TOFU, Tool: "pr --all-outlets",
				Command: fmt.Sprintf("musu-marketer pr %q -p %s --all-outlets", kw, project),
				Goal:    "매체 노출 초안 (검토 후 운영자 발송)"},
		)
		day += 7
	}

	// Standing funnel actions — the acquisition machine around the content.
	p.Actions = append(p.Actions,
		Action{Day: 1, Stage: MOFU, Tool: "opt-in",
			Command: "리드마그넷(무료 농지 진단 PDF) + 더블옵트인 구독 폼 게시",
			Goal:    "방문자 → 동의 기반 구독자 (정보통신망법 §50 준수)"},
		Action{Day: 3, Stage: MOFU, Tool: "nurture",
			Command: "nurikun 뉴스레터 시퀀스 준비 (human-in-the-loop 발송)",
			Goal:    "구독자 신뢰 형성, 재방문 유도"},
		Action{Day: 14, Stage: BOFU, Tool: "referral",
			Command: "musu-marketer referral (구독자 ↔ 파트너 변호사 매칭 초안)",
			Goal:    "고관여 구독자 → 상담/회원 전환 (검토 후 연결)"},
	)
	return p
}

// Render produces the campaign brief markdown.
func (p Plan) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("project: %q\n", p.Project))
	b.WriteString(fmt.Sprintf("target_members: %d\n", p.Config.TargetMembers))
	b.WriteString("kind: acquisition-funnel-plan\n")
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("# 회원 %d명 획득 캠페인 플랜 — %s\n\n", p.Config.TargetMembers, p.Project))

	b.WriteString("## 필요 규모 (가정 기반 — 실측으로 반드시 검증)\n\n")
	b.WriteString(fmt.Sprintf("- 목표 회원: **%d명**\n", p.Config.TargetMembers))
	b.WriteString(fmt.Sprintf("- 필요 구독자(opt-in): **%d명** (opt-in→회원 전환 %.0f%% 가정)\n",
		p.Math.OptInsNeeded, p.Config.MemberRate*100))
	b.WriteString(fmt.Sprintf("- 필요 방문자: **%s명** (방문→opt-in %.1f%% 가정)\n",
		commaInt(p.Math.VisitorsNeeded), p.Config.OptInRate*100))
	b.WriteString(fmt.Sprintf("- 월 필요 방문자: **%s명** (%d개월 horizon)\n",
		commaInt(p.Math.MonthlyVisits), horizonOr(p.Config.HorizonMonths)))
	b.WriteString(fmt.Sprintf("- 필요 SEO 글 수: **약 %d편** (글당 월 %d방문 × 페르소나 ×%.1f 가정)\n\n",
		p.Math.ArticlesNeeded, p.Config.VisitsPerArticleMo, math.Max(1, p.Config.PersonaMultiplier)))
	b.WriteString("> ⚠️ 위 수치는 가정입니다. GSC/GA 실측이 들어오면 opt-in율·글당 방문수를 갱신해 재계산하세요.\n\n")

	b.WriteString("## 캠페인 캘린더\n\n")
	b.WriteString("| Day | Stage | Tool | 액션 | 목표 |\n|---|---|---|---|---|\n")
	for _, a := range p.Actions {
		b.WriteString(fmt.Sprintf("| %d | %s | %s | `%s` | %s |\n",
			a.Day, a.Stage, a.Tool, a.Command, a.Goal))
	}
	b.WriteString("\n## 건강성 가드레일\n\n")
	b.WriteString("- 모든 콘텐츠는 Citation Gate 통과 (출처 없는 법조문 = 출하 불가)\n")
	b.WriteString("- 발송형(뉴스레터·PR·referral)은 전부 human-in-the-loop, 옵트인/동의 기반\n")
	b.WriteString("- 봇·가짜리뷰·페르소나 위조·공포압박 카피 금지\n")
	return b.String()
}

func horizonOr(h int) int {
	if h <= 0 {
		return 12
	}
	return h
}

// commaInt formats an int with thousands separators (no locale dep).
func commaInt(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
