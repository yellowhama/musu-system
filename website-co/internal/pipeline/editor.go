package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/website-co/internal/prompts"
)

// Verdict — 편집장 판정.
type Verdict struct {
	Decision string   `json:"decision"` // "approve" | "request_changes"
	Changes  []string `json:"changes"`
	Notes    string   `json:"notes"`
}

// Approved — 승인 여부.
func (v Verdict) Approved() bool { return strings.EqualFold(strings.TrimSpace(v.Decision), "approve") }

// Review — 초안을 검수하고 JSON 판정을 반환한다(YMYL 게이트).
func Review(ctx context.Context, r Runner, v prompts.Vars, draft string) (Verdict, error) {
	user := "## 초안\n\n" + draft + "\n\n위 초안을 검수하고 판정 JSON만 출력하라."
	out, err := r.Run(ctx, prompts.Editor(v), user)
	if err != nil {
		return Verdict{}, fmt.Errorf("편집장 패스: %w", err)
	}
	verdict, err := parseVerdict(out)
	if err != nil {
		return Verdict{}, fmt.Errorf("편집장 판정 파싱: %w (출력: %.200s)", err, out)
	}
	if !verdict.Approved() && verdict.Decision != "request_changes" {
		return Verdict{}, fmt.Errorf("편집장 decision 값 이상: %q", verdict.Decision)
	}
	return verdict, nil
}

// parseVerdict — 출력에서 첫 { ~ 마지막 } 사이 JSON을 추출해 파싱(군더더기 방어).
func parseVerdict(s string) (Verdict, error) {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end < start {
		return Verdict{}, fmt.Errorf("JSON 객체 없음")
	}
	var v Verdict
	if err := json.Unmarshal([]byte(s[start:end+1]), &v); err != nil {
		return Verdict{}, err
	}
	return v, nil
}
