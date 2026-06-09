package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/publish"
	"github.com/yellowhama/musu-system/website-co/internal/queue"
	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

const validDraft = `---
title: 테스트 기사
slug: test-article
category: 실전가이드
tags: [농지, 세금, 가이드, 소유자]
published: false
---

## 무엇을 알아야 하나
출처는 law.go.kr 기준이다.

## 왜 중요한가
중요하다.

## 내 농지에 미치는 영향
내 농지 상황에 따라 다르다.

## 오늘 확인할 것
- 확인할 것: 서류 점검.

## 출처
- https://law.go.kr
`

// fakeRunner — claude 없이 작가/편집장을 흉내낸다(시스템 프롬프트로 역할 구분).
type fakeRunner struct{}

func (fakeRunner) Run(_ context.Context, system, _ string) (string, error) {
	if !strings.Contains(system, "전담 작가") {
		return `{"decision":"approve","changes":[],"notes":"통과"}`, nil
	}
	return validDraft, nil // 작가
}

func TestDriveOne_PublishesAndCompletes(t *testing.T) {
	tmp := t.TempDir()
	cfg := tenant.Config{
		Site:      "test",
		Brand:     "테스트",
		DraftsDir: filepath.Join(tmp, "drafts"),
		MaxRounds: 3,
		Publish:   tenant.Publish{Mode: "file", Dir: filepath.Join(tmp, "pub")},
	}
	q, _ := queue.Open(filepath.Join(tmp, "q.json"))
	q.Add(queue.Item{Slug: "test-article", Topic: "테스트 토픽"})
	ledger, _ := publish.LoadLedger(filepath.Join(tmp, "ledger.json"))
	now := time.Unix(1700000000, 0)
	opt := Options{Publish: true, Now: func() time.Time { return now }}
	opt.defaults()

	it, ok := q.Claim(now, opt.Lease)
	if !ok {
		t.Fatal("claim 실패")
	}
	driveOne(context.Background(), cfg, fakeRunner{}, q, ledger, it, opt)

	// 발행 파일 존재
	if _, err := os.Stat(filepath.Join(tmp, "pub", "test-article.md")); err != nil {
		t.Fatalf("발행 파일 없음: %v", err)
	}
	// 원장 기록
	if !ledger.IsPublished("test-article") {
		t.Fatal("원장 미기록")
	}
	// 큐 done → pending 0
	if q.Pending() != 0 {
		t.Fatalf("pending=%d (기대 0)", q.Pending())
	}
}

// needs_human 경로: 편집장이 계속 반려 → 라운드 캡 → needs_human(절대 blocked 아님).
type rejectRunner struct{}

func (rejectRunner) Run(_ context.Context, system, _ string) (string, error) {
	if !strings.Contains(system, "전담 작가") {
		return `{"decision":"request_changes","changes":["출처 보강"],"notes":"보류"}`, nil
	}
	return validDraft, nil
}

func TestDriveOne_NeedsHumanNotBlocked(t *testing.T) {
	tmp := t.TempDir()
	cfg := tenant.Config{Site: "t", Brand: "t", DraftsDir: filepath.Join(tmp, "d"), MaxRounds: 2,
		Publish: tenant.Publish{Mode: "file", Dir: filepath.Join(tmp, "p")}}
	q, _ := queue.Open(filepath.Join(tmp, "q.json"))
	q.Add(queue.Item{Slug: "x", Topic: "T"})
	ledger, _ := publish.LoadLedger(filepath.Join(tmp, "l.json"))
	now := time.Unix(1700000000, 0)
	opt := Options{Publish: true, Now: func() time.Time { return now }}
	opt.defaults()
	it, _ := q.Claim(now, opt.Lease)
	driveOne(context.Background(), cfg, rejectRunner{}, q, ledger, it, opt)

	// needs_human으로 종료(blocked 같은 건 존재하지 않음 — 구조적)
	if q.Pending() != 0 {
		t.Fatalf("pending=%d (needs_human도 종료상태라 0이어야)", q.Pending())
	}
}
