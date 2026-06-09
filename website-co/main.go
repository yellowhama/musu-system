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
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/agent"
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
	timeout := fs.Duration("timeout", 5*time.Minute, "claude 호출 타임아웃")
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
	// Phase 1: 파이프라인 루프 배선 예정(T1.8). 현재는 토픽 에코.
	fmt.Fprintln(os.Stderr, "[stub] once --topic:", *topic, "— 파이프라인 배선 예정(T1.5~T1.9)")
}

func cmdRun(args []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Fprintln(os.Stderr, "[stub] run — 테넌트 스케줄러 미구현(Phase 3). Ctrl+C로 종료.")
	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "종료.")
}
