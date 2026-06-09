package queue

import (
	"path/filepath"
	"testing"
	"time"
)

func TestClaimLeaseComplete(t *testing.T) {
	q, err := Open(filepath.Join(t.TempDir(), "q.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Add(Item{Slug: "a", Topic: "A"}, Item{Slug: "b", Topic: "B"}); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0)
	lease := 30 * time.Minute

	// 첫 claim = a
	a, ok := q.Claim(now, lease)
	if !ok || a.Slug != "a" {
		t.Fatalf("claim 1 실패: %+v ok=%v", a, ok)
	}
	// 둘째 claim = b (a는 lease 보유 중이라 스킵)
	b, ok := q.Claim(now.Add(time.Minute), lease)
	if !ok || b.Slug != "b" {
		t.Fatalf("claim 2가 b 아님: %+v", b)
	}
	// 셋째 claim = 없음 (둘 다 in_progress, lease 유효)
	if _, ok := q.Claim(now.Add(2*time.Minute), lease); ok {
		t.Fatal("lease 유효한데 claim 됨(중복작업 위험)")
	}
	// a 완료
	if err := q.Complete("a", Done, 2, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if q.Pending() != 1 {
		t.Fatalf("pending=%d (기대 1: b만)", q.Pending())
	}
}

func TestLeaseExpiryReclaim(t *testing.T) {
	q, _ := Open(filepath.Join(t.TempDir(), "q.json"))
	q.Add(Item{Slug: "a", Topic: "A"})
	now := time.Unix(1700000000, 0)
	lease := 30 * time.Minute

	a, ok := q.Claim(now, lease)
	if !ok || a.Slug != "a" {
		t.Fatal("초기 claim 실패")
	}
	// lease 만료 전 = 재claim 불가(재시작 후에도 다른 워커 보호)
	if _, ok := q.Claim(now.Add(10*time.Minute), lease); ok {
		t.Fatal("lease 만료 전인데 재claim")
	}
	// lease 만료 후 = 재claim 가능(크래시/재시작 안전 — reconciler 불요로 block-storm 불가)
	a2, ok := q.Claim(now.Add(31*time.Minute), lease)
	if !ok || a2.Slug != "a" {
		t.Fatal("lease 만료 후 재claim 실패(재시작 안전 깨짐)")
	}
}
