package pipeline

import (
	"regexp"
	"strings"
)

// 필수 5섹션(발행 게이트 SSoT). 편집장·작가 프롬프트와 동일.
var requiredSections = []string{
	"## 무엇을 알아야 하나",
	"## 왜 중요한가",
	"## 내 농지에 미치는 영향",
	"## 오늘 확인할 것",
	"## 출처",
}

var tagsLineRe = regexp.MustCompile(`(?m)^tags:\s*\[(.*?)\]`)
var goKrRe = regexp.MustCompile(`\.go\.kr`)

// Validate — 초안에 발행 게이트 핵심 규칙을 적용해 위반 목록을 반환한다(빈 슬라이스=통과).
// paperclip content_validate publishGate의 Go 포팅(핵심 규칙). 완전판은 MCP 호출로 보강 가능(후속).
func Validate(draft string) []string {
	var fails []string
	for _, sec := range requiredSections {
		if !strings.Contains(draft, sec) {
			fails = append(fails, "필수 섹션 누락: "+sec)
		}
	}
	if strings.Contains(draft, "## 내 농지에 미치는 영향") && !strings.Contains(draft, "내 농지") {
		fails = append(fails, `"내 농지" 표현 누락(영향 섹션)`)
	}
	if !strings.Contains(draft, "확인할 것") {
		fails = append(fails, `"확인할 것" 체크리스트 표현 누락`)
	}
	// tags 4개 이상
	if m := tagsLineRe.FindStringSubmatch(draft); m != nil {
		n := 0
		for _, t := range strings.Split(m[1], ",") {
			if strings.TrimSpace(t) != "" {
				n++
			}
		}
		if n < 4 {
			fails = append(fails, "tags 4개 미만")
		}
	} else {
		fails = append(fails, "frontmatter tags 누락")
	}
	// 출처 .go.kr 1개 이상
	if !goKrRe.MatchString(draft) {
		fails = append(fails, "1차 출처(.go.kr) 누락")
	}
	// 환각 위험 금지어(단정)
	for _, w := range []string{"반드시", "확실히", "100%", "무조건"} {
		if strings.Contains(draft, w) {
			fails = append(fails, "단정 표현 사용: "+w)
		}
	}
	return fails
}
