package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/website-co/internal/prompts"
)

// Write — 토픽으로 기사 마크다운을 생성한다(첫 패스). 작가는 파일을 쓰지 않고 마크다운을 반환.
func Write(ctx context.Context, r Runner, v prompts.Vars, topic string) (string, error) {
	user := fmt.Sprintf("토픽: %s\n\n위 토픽으로 frontmatter부터 시작하는 완성 기사 마크다운만 출력하라.", topic)
	out, err := r.Run(ctx, prompts.Writer(v), user)
	if err != nil {
		return "", fmt.Errorf("작가 패스: %w", err)
	}
	return stripFences(out), nil
}

// Revise — 이전 초안 + 피드백으로 수정한다. **처음부터 재생성하지 말고 나열된 문제만 고쳐** 구조 회귀를 막는다.
// (섀도 런 실측: 매 재작성 시 전체 재생성하면 구조 섹션이 누락돼 수렴 실패 — R2 11검증실패.)
func Revise(ctx context.Context, r Runner, v prompts.Vars, topic, priorDraft string, feedback []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "토픽: %s\n\n", topic)
	b.WriteString("## 이전 초안\n\n")
	b.WriteString(priorDraft)
	b.WriteString("\n\n## 고칠 점 (아래만 정확히 고치고, 통과한 나머지 구조·섹션·frontmatter는 그대로 유지하라)\n")
	for i, c := range feedback {
		fmt.Fprintf(&b, "%d. %s\n", i+1, c)
	}
	b.WriteString("\n고친 **완성 기사 마크다운만** 출력하라(frontmatter부터). 5개 필수 섹션을 절대 빠뜨리지 마라.")

	out, err := r.Run(ctx, prompts.Writer(v), b.String())
	if err != nil {
		return "", fmt.Errorf("작가 수정 패스: %w", err)
	}
	return stripFences(out), nil
}

// stripFences — 모델이 실수로 ```markdown ... ``` 로 감싸면 벗긴다(출력 형식 방어).
func stripFences(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	// 첫 줄(``` 또는 ```markdown) 제거
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		t = t[i+1:]
	}
	t = strings.TrimSuffix(strings.TrimRight(t, " \n"), "```")
	return strings.TrimSpace(t)
}
