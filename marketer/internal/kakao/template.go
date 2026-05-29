// Package kakao implements the content layer of Sprint 2 — Kakao 알림톡
// (AlimTalk) template drafting. It is pure and deterministic (no LLM, no
// network) and intentionally STOPS before sending: 알림톡 templates must be
// registered and pre-approved by Kakao before any send, and the actual
// transport belongs to nurikun (v0.4.0 internal HTTP, not yet wired). What this
// package produces is reviewable template drafts plus a deterministic
// compliance check.
//
// Healthy-marketing scope: only 정보성 (informational/transactional) templates
// tied to a concrete user action (the consent basis). 광고성 (promotional,
// 친구톡) messaging requires separate marketing consent and is explicitly out of
// scope here — the validator rejects promotional markers so a promo message can
// never masquerade as a transactional one.
package kakao

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// CategoryInformational is the only category this package emits.
const CategoryInformational = "정보성"

// maxBodyRunes is Kakao's 알림톡 body limit (1,000 characters).
const maxBodyRunes = 1000

// Button is a template button. Type follows Kakao codes: WL = 웹링크.
type Button struct {
	Name string
	Type string
	URL  string
}

// Template is one 알림톡 template draft. Trigger is the user action that
// legitimizes the message (the 정보성/consent basis) and must be non-empty.
type Template struct {
	Code      string   // operator-facing template id, e.g. "njd_optin_confirm"
	Title     string   // human label
	Category  string   // must be CategoryInformational
	Trigger   string   // user action that fires it (consent basis)
	Body      string   // template text with #{변수} variables
	Variables []string // declared variables (without the #{} wrapper)
	Buttons   []Button
}

var varRe = regexp.MustCompile(`#\{([^}]+)\}`)

// promoMarkers are tokens that signal 광고성 content. Their presence in a
// "정보성" template is a compliance failure — promo must go through 친구톡 with
// separate consent, never disguised as transactional.
var promoMarkers = []string{"광고", "이벤트", "할인", "특가", "쿠폰", "프로모션", "세일", "최저가"}

// DefaultTemplates are NJD 정보성 templates: each is tied to a real user action
// (subscribe / request diagnosis / diagnosis ready), so it is consent-based.
var DefaultTemplates = []Template{
	{
		Code:      "njd_optin_confirm",
		Title:     "뉴스레터 구독 확인 (더블옵트인)",
		Category:  CategoryInformational,
		Trigger:   "사용자가 농지다 뉴스레터 구독을 신청함",
		Body:      "#{name}님, 농지다 뉴스레터 구독을 신청하셨습니다.\n아래 버튼을 눌러 구독을 확정해 주세요. 확정하지 않으면 발송되지 않습니다.",
		Variables: []string{"name", "token"},
		Buttons:   []Button{{Name: "구독 확정하기", Type: "WL", URL: "https://nongjida.kr/confirm?token=#{token}"}},
	},
	{
		Code:      "njd_diagnosis_received",
		Title:     "무료 농지 진단 접수 안내",
		Category:  CategoryInformational,
		Trigger:   "사용자가 무료 농지 진단을 신청함",
		Body:      "#{name}님, 신청하신 농지 진단이 정상 접수되었습니다.\n영업일 기준 #{days}일 이내에 결과를 안내드리겠습니다.",
		Variables: []string{"name", "days"},
	},
	{
		Code:      "njd_diagnosis_ready",
		Title:     "농지 진단 결과 안내",
		Category:  CategoryInformational,
		Trigger:   "사용자가 신청한 진단 결과가 준비됨",
		Body:      "#{name}님, 신청하신 농지 진단 결과가 준비되었습니다.\n아래 버튼에서 결과를 확인하실 수 있습니다.",
		Variables: []string{"name", "id"},
		Buttons:   []Button{{Name: "진단 결과 확인", Type: "WL", URL: "https://nongjida.kr/diagnosis/#{id}"}},
	},
}

// Issue is one compliance problem found on a template.
type Issue struct {
	Code    string
	Problem string
}

// Validate runs the deterministic 알림톡 compliance checks and returns any
// issues. An empty slice means the template is registration-ready.
func Validate(t Template) []Issue {
	var issues []Issue
	add := func(p string) { issues = append(issues, Issue{Code: t.Code, Problem: p}) }

	if t.Category != CategoryInformational {
		add(fmt.Sprintf("카테고리가 %q — 정보성만 허용 (광고성은 친구톡+별도 동의)", t.Category))
	}
	if strings.TrimSpace(t.Trigger) == "" {
		add("트리거(사용자 행동) 없음 — 정보성 메시지는 사용자 행동/동의에 근거해야 함")
	}
	if n := len([]rune(t.Body)); n > maxBodyRunes {
		add(fmt.Sprintf("본문 %d자 — 1,000자 제한 초과", n))
	}
	// Promo markers must not appear in a 정보성 template.
	for _, m := range promoMarkers {
		if strings.Contains(t.Body, m) {
			add(fmt.Sprintf("광고성 표현 %q 포함 — 정보성 템플릿에 사용 불가", m))
		}
	}
	// Every #{var} used in body or button URLs must be declared, and vice versa.
	declared := map[string]bool{}
	for _, v := range t.Variables {
		declared[v] = true
	}
	used := map[string]bool{}
	for _, m := range varRe.FindAllStringSubmatch(t.Body, -1) {
		used[m[1]] = true
	}
	for _, b := range t.Buttons {
		for _, m := range varRe.FindAllStringSubmatch(b.URL, -1) {
			used[m[1]] = true
		}
	}
	for v := range used {
		if !declared[v] {
			add(fmt.Sprintf("변수 #{%s} 사용했으나 Variables에 미선언", v))
		}
	}
	for v := range declared {
		if !used[v] {
			add(fmt.Sprintf("변수 %q 선언했으나 본문/버튼에서 미사용", v))
		}
	}
	return issues
}

// Render produces the registration-ready draft document for a set of templates.
// It makes the boundaries explicit: pre-approval required, send via nurikun
// (human-in-the-loop), 광고성 out of scope.
func Render(templates []Template) string {
	var b strings.Builder
	b.WriteString("---\nkind: kakao-alimtalk-templates\ncategory: 정보성\n")
	b.WriteString("delivery: nurikun v0.4.0 (미연동) · human-in-the-loop\n---\n\n")
	b.WriteString("# 카카오 알림톡 템플릿 초안 (정보성)\n\n")
	b.WriteString("> ⚠️ 초안입니다. 카카오 비즈메시지에 **사전 등록·심사 승인** 후에만 발송 가능합니다. " +
		"발송 전송은 nurikun(v0.4.0 예정)이 옵트인 구독자에게 human-in-the-loop로 처리합니다. " +
		"**광고성(친구톡) 메시지는 별도 마케팅 수신동의가 필요하며 여기 범위 밖입니다.**\n\n")

	for _, t := range templates {
		b.WriteString(fmt.Sprintf("## %s (`%s`)\n\n", t.Title, t.Code))
		b.WriteString(fmt.Sprintf("- 카테고리: %s\n", t.Category))
		b.WriteString(fmt.Sprintf("- 발송 근거(트리거): %s\n", t.Trigger))
		if len(t.Variables) > 0 {
			vs := append([]string(nil), t.Variables...)
			sort.Strings(vs)
			b.WriteString(fmt.Sprintf("- 변수: %s\n", strings.Join(vs, ", ")))
		}
		b.WriteString("\n```\n")
		b.WriteString(t.Body)
		b.WriteString("\n```\n")
		for _, btn := range t.Buttons {
			b.WriteString(fmt.Sprintf("- [버튼] %s (%s) → %s\n", btn.Name, btn.Type, btn.URL))
		}
		if issues := Validate(t); len(issues) > 0 {
			b.WriteString("\n**⚠️ 컴플라이언스 이슈:**\n")
			for _, is := range issues {
				b.WriteString(fmt.Sprintf("- %s\n", is.Problem))
			}
		} else {
			b.WriteString("\n✅ 컴플라이언스 통과 (등록 준비 완료)\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
