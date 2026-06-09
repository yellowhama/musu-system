// Package pipeline — 콘텐츠 파이프라인(작가→검증→편집장→수정루프). claude는 stateless 실행기.
// 상태(라운드·초안·판정)는 호출자가 소유 → paperclip 이슈/reconciler 무관(block-storm 원천 차단).
package pipeline

import "context"

// Runner — claude 실행기 추상(agent.Client가 구현). 테스트 시 페이크 주입.
type Runner interface {
	Run(ctx context.Context, system, user string) (string, error)
}
