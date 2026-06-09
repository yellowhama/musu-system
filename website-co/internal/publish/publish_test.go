package publish

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

const sampleDraft = `---
title: 농지연금 가이드
slug: farmland-pension-guide
category: 실전가이드
tags: [농지연금, 노후, 농어촌공사, 가이드]
published: false
---

## 무엇을 알아야 하나
내용.
`

func TestParseFrontmatter(t *testing.T) {
	m, err := ParseFrontmatter(sampleDraft)
	if err != nil {
		t.Fatal(err)
	}
	if m.Slug != "farmland-pension-guide" || m.Title != "농지연금 가이드" || m.Category != "실전가이드" {
		t.Fatalf("파싱 불일치: %+v", m)
	}
}

func TestParseFrontmatterMissing(t *testing.T) {
	if _, err := ParseFrontmatter("## 본문만 있음"); err == nil {
		t.Fatal("slug/title 누락인데 에러 안 남")
	}
}

func TestPublishFileMode_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	cfg := tenant.Config{
		Site:      "test",
		Brand:     "테스트",
		DraftsDir: filepath.Join(tmp, "drafts"),
		Publish:   tenant.Publish{Mode: "file", Dir: filepath.Join(tmp, "pub")},
	}
	ledger, err := LoadLedger(filepath.Join(tmp, "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0)

	// 1차 발행
	r, err := Publish(context.Background(), cfg, sampleDraft, ledger, now)
	if err != nil {
		t.Fatal(err)
	}
	if r.Skipped {
		t.Fatal("1차인데 skip됨")
	}
	if _, err := os.Stat(filepath.Join(tmp, "pub", "farmland-pension-guide.md")); err != nil {
		t.Fatalf("발행 파일 없음: %v", err)
	}
	if !ledger.IsPublished("farmland-pension-guide") {
		t.Fatal("원장에 published 미기록")
	}

	// 2차 발행 = 멱등 skip
	r2, err := Publish(context.Background(), cfg, sampleDraft, ledger, now)
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Skipped {
		t.Fatal("2차인데 skip 안 됨(중복발행 위험)")
	}
}
