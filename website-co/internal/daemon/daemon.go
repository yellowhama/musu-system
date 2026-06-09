// Package daemon — 테넌트 큐를 연속 처리하는 운영 루프. paperclip 서버/reconciler 없음.
// 토픽 claim → 작가/편집장 루프 → 발행(또는 needs_human) → Complete. lease로 재시작 안전.
package daemon

import (
	"context"
	"log"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/pipeline"
	"github.com/yellowhama/musu-system/website-co/internal/prompts"
	"github.com/yellowhama/musu-system/website-co/internal/publish"
	"github.com/yellowhama/musu-system/website-co/internal/queue"
	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

// Options — 데몬 동작 설정.
type Options struct {
	Concurrency int           // 동시 드라이브 워커 수(기본 2)
	Lease       time.Duration // 토픽 임차 시간(기본 45분)
	Poll        time.Duration // 큐 빈 때 폴 간격(기본 30초)
	Publish     bool          // 승인 시 실제 발행(false=섀도)
	Now         func() time.Time
}

func (o *Options) defaults() {
	if o.Concurrency <= 0 {
		o.Concurrency = 2
	}
	if o.Lease == 0 {
		o.Lease = 45 * time.Minute
	}
	if o.Poll == 0 {
		o.Poll = 30 * time.Second
	}
	if o.Now == nil {
		o.Now = time.Now
	}
}

// RunTenant — 한 테넌트의 큐를 ctx 종료까지 연속 처리한다.
func RunTenant(ctx context.Context, cfg tenant.Config, cl pipeline.Runner, opt Options) error {
	opt.defaults()
	q, err := queue.Open(cfg.BacklogPath)
	if err != nil {
		return err
	}
	ledger, err := publish.LoadLedger(cfg.LedgerPath)
	if err != nil {
		return err
	}
	sem := make(chan struct{}, opt.Concurrency)
	log.Printf("[%s] 데몬 시작 (동시 %d, lease %s, 발행=%v)", cfg.Site, opt.Concurrency, opt.Lease, opt.Publish)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		item, ok := q.Claim(opt.Now(), opt.Lease)
		if !ok {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(opt.Poll):
			}
			continue
		}
		sem <- struct{}{}
		go func(it queue.Item) {
			defer func() { <-sem }()
			driveOne(ctx, cfg, cl, q, ledger, it, opt)
		}(item)
	}
}

func driveOne(ctx context.Context, cfg tenant.Config, cl pipeline.Runner, q *queue.Queue, ledger *publish.Ledger, it queue.Item, opt Options) {
	v := prompts.Vars{Brand: cfg.Brand, Slug: it.Slug}
	topic := it.Topic
	if topic == "" {
		topic = it.Slug
	}
	res, err := pipeline.Drive(ctx, cl, v, topic, cfg.MaxRounds)
	if err != nil {
		log.Printf("[%s] %s 드라이브 오류: %v", cfg.Site, it.Slug, err)
		// lease 만료 후 재claim되게 둠(상태 변경 안 함). 영구실패는 향후 attempt 카운트로.
		return
	}
	if res.Status != pipeline.StatusApproved {
		log.Printf("[%s] %s → needs_human (%d라운드): %s", cfg.Site, it.Slug, res.Rounds, res.Verdict.Notes)
		_ = q.Complete(it.Slug, queue.NeedsHuman, res.Rounds, opt.Now())
		return
	}
	if !opt.Publish {
		log.Printf("[%s] %s → 승인(섀도, 발행 안 함, %d라운드)", cfg.Site, it.Slug, res.Rounds)
		_ = q.Complete(it.Slug, queue.Done, res.Rounds, opt.Now())
		return
	}
	pr, err := publish.Publish(ctx, cfg, res.Draft, ledger, opt.Now())
	if err != nil {
		log.Printf("[%s] %s 발행 오류: %v", cfg.Site, it.Slug, err)
		return // 재시도 가능(lease 만료 후)
	}
	if pr.Skipped {
		log.Printf("[%s] %s 이미 발행됨(멱등)", cfg.Site, it.Slug)
	} else {
		log.Printf("[%s] ✅ %s 발행: %s", cfg.Site, pr.Slug, pr.URL)
	}
	_ = q.Complete(it.Slug, queue.Done, res.Rounds, opt.Now())
}
