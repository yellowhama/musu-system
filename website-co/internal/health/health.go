// Package health — IETF health+json 엔드포인트. 무인 운영의 거짓건강 방지(관측성).
// 요청마다 테넌트 상태 파일(큐·원장)을 읽어 보고한다(공유 상태 불요, 단순).
package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/publish"
	"github.com/yellowhama/musu-system/website-co/internal/queue"
	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

// TenantStatus — 테넌트 1개 상태.
type TenantStatus struct {
	Site           string `json:"site"`
	Pending        int    `json:"pending"`
	PublishedToday int    `json:"publishedToday"`
	PublishedTotal int    `json:"publishedTotal"`
	DailyCap       int    `json:"dailyCap"`
}

// Handler — /health 핸들러. 테넌트 목록을 클로저로 캡처.
func Handler(tenants []tenant.Config, now func() time.Time) http.HandlerFunc {
	if now == nil {
		now = time.Now
	}
	return func(w http.ResponseWriter, r *http.Request) {
		out := map[string]any{"status": "pass", "service": "website-co"}
		var ts []TenantStatus
		status := "pass"
		for _, c := range tenants {
			s := TenantStatus{Site: c.Site, DailyCap: c.DailyCap}
			if q, err := queue.Open(c.BacklogPath); err == nil {
				s.Pending = q.Pending()
			}
			if l, err := publish.LoadLedger(c.LedgerPath); err == nil {
				s.PublishedToday = l.PublishedToday(now())
				s.PublishedTotal = len(l.Published)
			}
			if s.Pending == 0 {
				status = "warn" // 큐 고갈 임박
			}
			ts = append(ts, s)
		}
		out["status"] = status
		out["tenants"] = ts
		w.Header().Set("Content-Type", "application/health+json")
		if status != "pass" {
			w.WriteHeader(http.StatusOK) // health+json은 200 유지, status 필드로 표현
		}
		_ = json.NewEncoder(w).Encode(out)
	}
}
