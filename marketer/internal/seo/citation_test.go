package seo

import (
	"testing"

	"github.com/yellowhama/musu-system/marketer/internal/bridge"
)

func testPack() SourcePack {
	return NewSourcePack("농지법 처분명령", []bridge.KnowledgeSource{
		{ID: "law-1", Title: "농지법 제10조", Source: "법제처", Content: "농지 처분의무"},
		{ID: "law-2", Title: "2026 농지 전수조사", Source: "농림축산식품부", Content: "전수조사 개요"},
	})
}

func TestSourcePackLabels(t *testing.T) {
	p := testPack()
	if got := p.Labels(); len(got) != 2 || got[0] != "S1" || got[1] != "S2" {
		t.Fatalf("labels = %v, want [S1 S2]", got)
	}
	if !p.HasLabel("S1") || p.HasLabel("S3") {
		t.Fatalf("HasLabel wrong: S1=%v S3=%v", p.HasLabel("S1"), p.HasLabel("S3"))
	}
}

func TestCheckCitations_CitedClaimPasses(t *testing.T) {
	body := "농지법 제10조에 따라 처분의무가 발생한다 [S1].\n2026년 농지 전수조사가 시행된다 [S2]."
	rep := CheckCitations(body, testPack())
	if rep.CitedClaims != 2 {
		t.Errorf("CitedClaims = %d, want 2", rep.CitedClaims)
	}
	if len(rep.UncitedClaims) != 0 {
		t.Errorf("UncitedClaims = %v, want none", rep.UncitedClaims)
	}
	if !rep.OK(true) {
		t.Errorf("expected OK in strict mode")
	}
}

func TestCheckCitations_UncitedClaimBlockedInStrict(t *testing.T) {
	// A clear legal claim with no citation — the canonical "hallucinated law" risk.
	body := "농지법 제10조에 따라 1년 안에 처분명령이 내려진다."
	rep := CheckCitations(body, testPack())
	if len(rep.UncitedClaims) != 1 {
		t.Fatalf("UncitedClaims = %v, want 1", rep.UncitedClaims)
	}
	if rep.OK(true) {
		t.Errorf("strict mode must NOT pass an uncited legal claim")
	}
	if !rep.OK(false) {
		t.Errorf("non-strict mode should pass (warning only)")
	}
	if rep.Err(true) == nil {
		t.Errorf("expected blocking error in strict mode")
	}
}

func TestCheckCitations_InvalidLabelAlwaysFatal(t *testing.T) {
	// S9 does not exist in the pack -> fabricated citation, fatal even non-strict.
	body := "농지법 제10조 처분명령 관련 내용 [S9]."
	rep := CheckCitations(body, testPack())
	if len(rep.InvalidLabels) != 1 || rep.InvalidLabels[0] != "S9" {
		t.Fatalf("InvalidLabels = %v, want [S9]", rep.InvalidLabels)
	}
	if rep.OK(false) {
		t.Errorf("invalid label must fail even in non-strict mode")
	}
}

func TestCheckCitations_MarketingSentenceExempt(t *testing.T) {
	// No legal/numeric/authority signal -> not a claim -> no citation required.
	body := "농지 고민, 이제 농지다와 함께 해결하세요. 복잡한 문제도 쉽게 안내합니다."
	rep := CheckCitations(body, testPack())
	if len(rep.UncitedClaims) != 0 {
		t.Errorf("marketing prose should not require citations, got %v", rep.UncitedClaims)
	}
	if !rep.OK(true) {
		t.Errorf("expected OK")
	}
}

func TestCheckCitations_HeadingsAndFencesNotClaims(t *testing.T) {
	body := "## 농지법 제10조 개요\n\n```\n농지법 제10조 코드 예시\n```\n\n농지법 제10조는 처분의무를 규정한다 [S1]."
	rep := CheckCitations(body, testPack())
	if len(rep.UncitedClaims) != 0 {
		t.Errorf("headings/code fences must not be claims, got %v", rep.UncitedClaims)
	}
	if rep.CitedClaims != 1 {
		t.Errorf("CitedClaims = %d, want 1", rep.CitedClaims)
	}
}

func TestCheckCitations_InvalidLabelInHeadingCaught(t *testing.T) {
	// Fabricated label anywhere (even a heading) is still fatal.
	body := "## 개요 [S7]\n\n본문 [S1]."
	rep := CheckCitations(body, testPack())
	if len(rep.InvalidLabels) != 1 || rep.InvalidLabels[0] != "S7" {
		t.Fatalf("InvalidLabels = %v, want [S7]", rep.InvalidLabels)
	}
}
