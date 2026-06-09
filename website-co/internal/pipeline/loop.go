package pipeline

import (
	"context"
	"fmt"

	"github.com/yellowhama/musu-system/website-co/internal/prompts"
)

// Status — 루프 결과 상태.
type Status string

const (
	StatusApproved   Status = "approved"    // 편집장 승인 → 발행 가능
	StatusNeedsHuman Status = "needs_human" // 라운드 캡 내 미수렴 → 사람 검토(blocked 아님!)
)

// Result — 한 토픽의 파이프라인 결과. 상태를 호출자가 소유(paperclip 이슈 없음).
type Result struct {
	Topic   string
	Status  Status
	Draft   string
	Verdict Verdict
	Rounds  int
	Trace   []string // 라운드별 무슨 일이 있었는지(관측)
}

// Drive — 토픽을 작가→검증→편집장→(수정루프)로 완주시킨다. maxRounds 초과 = needs_human.
func Drive(ctx context.Context, r Runner, v prompts.Vars, topic string, maxRounds int) (Result, error) {
	if maxRounds <= 0 {
		maxRounds = 4
	}
	res := Result{Topic: topic}
	var feedback []string
	for round := 1; round <= maxRounds; round++ {
		res.Rounds = round
		// 1) 작가 패스
		draft, err := Write(ctx, r, v, topic, feedback)
		if err != nil {
			return res, err
		}
		res.Draft = draft

		// 2) 기계 검증 게이트
		if fails := Validate(draft); len(fails) > 0 {
			feedback = fails
			res.Trace = append(res.Trace, fmt.Sprintf("R%d 검증 실패 %d건 → 작가 재작성", round, len(fails)))
			continue
		}

		// 3) 편집장 패스(YMYL)
		verdict, err := Review(ctx, r, v, draft)
		if err != nil {
			return res, err
		}
		res.Verdict = verdict
		if verdict.Approved() {
			res.Status = StatusApproved
			res.Trace = append(res.Trace, fmt.Sprintf("R%d 편집장 승인", round))
			return res, nil
		}
		feedback = verdict.Changes
		res.Trace = append(res.Trace, fmt.Sprintf("R%d 편집장 변경요청 %d건 → 작가 재작성", round, len(verdict.Changes)))
	}
	res.Status = StatusNeedsHuman
	res.Trace = append(res.Trace, fmt.Sprintf("라운드 캡(%d) 초과 → needs_human", maxRounds))
	return res, nil
}
