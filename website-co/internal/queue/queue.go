// Package queue — 토픽 큐. paperclip "이슈"를 대체한다. 데몬이 단일 소유 → 외부 reconciler 없음
// → block-storm 구조적 불가. in_progress는 우리 상태이고 lease(타임아웃)로 재시작 안전을 보장한다.
package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Status — 토픽 상태. (paperclip과 달리 blocked 없음 — 미수렴은 needs_human.)
type Status string

const (
	Todo       Status = ""            // 미착수(빈 값 = 기존 backlog 호환)
	InProgress Status = "in_progress" // 작업 중(lease 보유)
	Done       Status = "done"        // 발행 완료
	NeedsHuman Status = "needs_human" // 라운드 캡 미수렴 — 사람 검토
	Failed     Status = "failed"      // 영구 실패
)

// Item — 큐 항목(기존 topics-backlog.json 호환: slug/topic/category/status).
type Item struct {
	Slug       string `json:"slug"`
	Topic      string `json:"topic"`
	Category   string `json:"category,omitempty"`
	DemandNote string `json:"demand_note,omitempty"`
	Status     Status `json:"status,omitempty"`
	LeasedAt   string `json:"leasedAt,omitempty"` // RFC3339, in_progress 임차 시각
	Rounds     int    `json:"rounds,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

// Queue — file 백엔드 토픽 큐. 데몬이 단일 소유(in-process mutex + 원자 쓰기).
type Queue struct {
	path  string
	mu    sync.Mutex
	items []Item
}

// Open — 큐 파일을 연다(없으면 빈 큐).
func Open(path string) (*Queue, error) {
	q := &Queue{path: path}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return q, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &q.items); err != nil {
		return nil, fmt.Errorf("큐 파싱(%s): %w", path, err)
	}
	return q, nil
}

// Claim — todo(또는 lease 만료된 in_progress) 1개를 in_progress로 임차해 반환한다.
// lease 만료 재claim = 재시작/크래시 안전(reconciler 불요). 없으면 (nil, false).
func (q *Queue) Claim(now time.Time, lease time.Duration) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.items {
		it := &q.items[i]
		switch it.Status {
		case Todo:
		case InProgress:
			if t, err := time.Parse(time.RFC3339, it.LeasedAt); err == nil && now.Sub(t) < lease {
				continue // 유효 lease — 다른 워커가 작업 중
			}
			// lease 만료 → 재claim 가능
		default:
			continue // done/needs_human/failed
		}
		it.Status = InProgress
		it.LeasedAt = now.UTC().Format(time.RFC3339)
		it.UpdatedAt = it.LeasedAt
		snapshot := *it
		_ = q.save()
		return snapshot, true
	}
	return Item{}, false
}

// Complete — 토픽을 종료 상태(done/needs_human/failed)로 마킹한다.
func (q *Queue) Complete(slug string, st Status, rounds int, now time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.items {
		if q.items[i].Slug == slug {
			q.items[i].Status = st
			q.items[i].Rounds = rounds
			q.items[i].UpdatedAt = now.UTC().Format(time.RFC3339)
			q.items[i].LeasedAt = ""
			return q.save()
		}
	}
	return fmt.Errorf("큐에 slug 없음: %s", slug)
}

// Add — 신규 토픽 추가(slug 중복은 스킵).
func (q *Queue) Add(items ...Item) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	have := map[string]bool{}
	for _, it := range q.items {
		have[it.Slug] = true
	}
	n := 0
	for _, it := range items {
		if it.Slug == "" || have[it.Slug] {
			continue
		}
		q.items = append(q.items, it)
		have[it.Slug] = true
		n++
	}
	if n > 0 {
		return n, q.save()
	}
	return 0, nil
}

// Pending — 미처리(todo+in_progress) 수.
func (q *Queue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, it := range q.items {
		if it.Status == Todo || it.Status == InProgress {
			n++
		}
	}
	return n
}

func (q *Queue) save() error {
	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(q.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := q.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, q.path)
}
