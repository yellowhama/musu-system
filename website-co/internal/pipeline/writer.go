package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/yellowhama/musu-system/website-co/internal/prompts"
)

// Write — 토픽(+선택 편집장 피드백)으로 기사 마크다운을 생성한다.
// 작가는 파일을 쓰지 않고 마크다운을 반환 → 호출자가 저장/발행을 소유.
func Write(ctx context.Context, r Runner, v prompts.Vars, topic string, feedback []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "토픽: %s\n", topic)
	if len(feedback) > 0 {
		b.WriteString("\n## 편집장 변경요청 (이 지시를 정확히 반영해 전체 기사를 다시 써라)\n")
		for i, c := range feedback {
			fmt.Fprintf(&b, "%d. %s\n", i+1, c)
		}
	}
	b.WriteString("\n위 토픽으로 frontmatter부터 시작하는 완성 기사 마크다운만 출력하라.")

	out, err := r.Run(ctx, prompts.Writer(v), b.String())
	if err != nil {
		return "", fmt.Errorf("작가 패스: %w", err)
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
