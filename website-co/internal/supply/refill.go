// Package supply — 토픽 공급(백로그 보충). 무인 시스템이 큐 고갈로 멈추지 않게 한다.
// 1차: 테넌트 큐레이트 풀(pool.json)에서 보충(안전망). 동적 발굴(마케터 MCP/naver_demand)은 후속.
package supply

import (
	"encoding/json"
	"os"

	"github.com/yellowhama/musu-system/website-co/internal/queue"
)

// PoolItem — 큐레이트 풀 항목.
type PoolItem struct {
	Slug       string `json:"slug"`
	Topic      string `json:"topic"`
	Category   string `json:"category,omitempty"`
	DemandNote string `json:"demand_note,omitempty"`
}

// Refill — pending < min 이면 풀에서 target까지 큐를 보충한다. 보충 건수 반환.
func Refill(q *queue.Queue, poolPath string, min, target int) (int, error) {
	if q.Pending() >= min {
		return 0, nil
	}
	b, err := os.ReadFile(poolPath)
	if err != nil {
		return 0, err
	}
	var pool []PoolItem
	if err := json.Unmarshal(b, &pool); err != nil {
		return 0, err
	}
	var add []queue.Item
	for _, p := range pool {
		if q.Pending()+len(add) >= target {
			break
		}
		add = append(add, queue.Item{Slug: p.Slug, Topic: p.Topic, Category: p.Category, DemandNote: p.DemandNote})
	}
	return q.Add(add...)
}
