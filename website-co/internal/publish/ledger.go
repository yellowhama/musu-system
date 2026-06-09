package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry — 발행 원장 항목(기존 published-approved.json 호환).
type Entry struct {
	At     string `json:"at"`               // RFC3339
	URL    string `json:"url,omitempty"`
	Title  string `json:"title,omitempty"`
	Intent bool   `json:"intent,omitempty"` // write-ahead: 발행 시작했으나 미완(중복방지)
}

// Ledger — slug→Entry. published-approved.json 형식: {"published": {...}}.
type Ledger struct {
	path      string
	Published map[string]Entry `json:"published"`
}

// LoadLedger — 원장을 읽는다(없으면 빈 원장).
func LoadLedger(path string) (*Ledger, error) {
	l := &Ledger{path: path, Published: map[string]Entry{}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	var raw struct {
		Published map[string]Entry `json:"published"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("원장 파싱: %w", err)
	}
	if raw.Published != nil {
		l.Published = raw.Published
	}
	return l, nil
}

// IsPublished — 이미 발행 완료(intent 아님)된 slug인가.
func (l *Ledger) IsPublished(slug string) bool {
	e, ok := l.Published[slug]
	return ok && !e.Intent
}

// MarkIntent — 발행 직전 write-ahead 의도 기록(중복발행 방지).
func (l *Ledger) MarkIntent(slug, title string, now time.Time) error {
	l.Published[slug] = Entry{At: now.UTC().Format(time.RFC3339), Title: title, Intent: true}
	return l.save()
}

// MarkPublished — 발행 성공 확정.
func (l *Ledger) MarkPublished(slug, title, url string, now time.Time) error {
	l.Published[slug] = Entry{At: now.UTC().Format(time.RFC3339), Title: title, URL: url}
	return l.save()
}

func (l *Ledger) save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	out := map[string]any{"published": l.Published}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.path) // 원자 교체
}
