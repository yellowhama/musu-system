// Command website-co — AI 웹사이트 운영회사, 단일 멀티테넌트 Go 데몬.
// work를 paperclip 이슈로 두지 않고(reconciler 무관, block-storm 원천 차단) 데몬이 상태를 소유하고
// claude를 stateless 실행기로 사용한다. PRD: musu-for-njd/docs/specs/musu-website-co-PRD.md
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/agent"
	"github.com/yellowhama/musu-system/website-co/internal/daemon"
	"github.com/yellowhama/musu-system/website-co/internal/pipeline"
	"github.com/yellowhama/musu-system/website-co/internal/prompts"
	"github.com/yellowhama/musu-system/website-co/internal/publish"
	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "once":
		cmdOnce(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `website-co — AI 웹사이트 운영회사 데몬
사용:
  website-co once --probe              claude 실행기 연결 점검(짧은 프롬프트 1회)
  website-co once --topic "<토픽>"     토픽 1개를 작가→편집장→(섀도) 1회전
  website-co run                       상시 데몬(테넌트 스케줄 구동) [미구현 stub]`)
}

func cmdOnce(args []string) {
	fs := flag.NewFlagSet("once", flag.ExitOnError)
	probe := fs.Bool("probe", false, "claude 실행기 연결 점검")
	topic := fs.String("topic", "", "드라이브할 토픽")
	timeout := fs.Duration("timeout", 8*time.Minute, "claude 호출 타임아웃")
	rounds := fs.Int("rounds", 4, "수정 루프 최대 라운드")
	tenantPath := fs.String("tenant", "", "테넌트 config 경로 또는 디렉토리(brand·발행·원장)")
	doPublish := fs.Bool("publish", false, "승인 시 실제 발행(미지정=섀도, 발행 안 함)")
	_ = fs.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cl := agent.New(agent.Options{Timeout: *timeout})

	if *probe {
		out, err := cl.Run(ctx, "You are a terse assistant. Reply in Korean.", "한 문장으로: 너는 누구냐?")
		if err != nil {
			fmt.Fprintln(os.Stderr, "probe 실패:", err)
			os.Exit(1)
		}
		fmt.Println("probe OK →", out)
		return
	}
	if *topic == "" {
		fmt.Fprintln(os.Stderr, "--probe 또는 --topic 필요")
		os.Exit(2)
	}

	// 테넌트 로드(있으면). 없으면 농지다 기본 brand로 섀도.
	brand, maxRounds := "농지다", *rounds
	var cfg *tenant.Config
	if *tenantPath != "" {
		c, err := tenant.Load(resolveTenant(*tenantPath))
		if err != nil {
			fmt.Fprintln(os.Stderr, "테넌트 로드 실패:", err)
			os.Exit(1)
		}
		cfg = &c
		brand, maxRounds = c.Brand, c.MaxRounds
	}

	res, err := pipeline.Drive(ctx, cl, prompts.Vars{Brand: brand}, *topic, maxRounds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "드라이브 실패:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "\n===== 결과: %s (%d라운드) =====\n", res.Status, res.Rounds)
	for _, t := range res.Trace {
		fmt.Fprintln(os.Stderr, " ·", t)
	}
	if res.Verdict.Notes != "" {
		fmt.Fprintln(os.Stderr, "편집장 노트:", res.Verdict.Notes)
	}

	// 승인 + --publish + 테넌트 = 실제 발행. 그 외 = 섀도.
	if res.Status == pipeline.StatusApproved && *doPublish && cfg != nil {
		lp := cfg.LedgerPath
		if lp == "" {
			lp = filepath.Join(filepath.Dir(resolveTenant(*tenantPath)), "state", "published-approved.json")
		}
		ledger, err := publish.LoadLedger(lp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "원장 로드 실패:", err)
			os.Exit(1)
		}
		pr, err := publish.Publish(ctx, *cfg, res.Draft, ledger, time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "발행 실패:", err)
			os.Exit(1)
		}
		if pr.Skipped {
			fmt.Fprintln(os.Stderr, "이미 발행됨(멱등):", pr.Slug)
		} else {
			fmt.Fprintf(os.Stderr, "✅ 발행 완료: %s → %s\n", pr.Slug, pr.URL)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "\n===== 초안(섀도, 발행 안 함) =====")
	fmt.Println(res.Draft)
}

// resolveTenant — config.json 경로 또는 디렉토리(또는 tenants/<name>)를 config.json 경로로 정규화.
func resolveTenant(p string) string {
	if strings.HasSuffix(p, ".json") {
		return p
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return filepath.Join(p, "config.json")
	}
	return filepath.Join("tenants", p, "config.json")
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	tenantsDir := fs.String("tenants", "tenants", "테넌트 디렉토리")
	concurrency := fs.Int("concurrency", 2, "테넌트별 동시 드라이브 워커")
	doPublish := fs.Bool("publish", false, "승인 시 실제 발행(미지정=섀도)")
	timeout := fs.Duration("timeout", 8*time.Minute, "claude 호출 타임아웃")
	_ = fs.Parse(args)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	tenants, err := tenant.LoadAll(*tenantsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "테넌트 로드 실패:", err)
		os.Exit(1)
	}
	if len(tenants) == 0 {
		fmt.Fprintln(os.Stderr, "테넌트 없음:", *tenantsDir)
		os.Exit(1)
	}
	cl := agent.New(agent.Options{Timeout: *timeout})
	opt := daemon.Options{Concurrency: *concurrency, Publish: *doPublish}

	var wg sync.WaitGroup
	for _, c := range tenants {
		wg.Add(1)
		go func(c tenant.Config) {
			defer wg.Done()
			if err := daemon.RunTenant(ctx, c, cl, opt); err != nil && ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "[%s] 데몬 종료: %v\n", c.Site, err)
			}
		}(c)
	}
	fmt.Fprintf(os.Stderr, "website-co 데몬 — %d 테넌트 구동(발행=%v). Ctrl+C로 종료.\n", len(tenants), *doPublish)
	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "종료 신호 — 진행 중 작업 마무리 대기...")
	wg.Wait()
	fmt.Fprintln(os.Stderr, "종료 완료.")
}
