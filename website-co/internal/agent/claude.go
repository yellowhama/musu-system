// Package agent — claude CLI를 stateless 실행기로 감싼다.
// 핵심 견고성: ① PATH 주입(%APPDATA%\npm — claude.cmd 위치, 무인 환경 PATH 상실 버그 방지)
//   ② 프롬프트는 stdin으로 전달(긴 기사 전문도 OS arg 길이 제한 무관) ③ 타임아웃 + 지수백오프 재시도.
// claude는 파일을 쓰지 않고 결과를 stdout으로만 반환 → 호출자가 모든 상태·파일·발행을 소유.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Options — 실행기 설정.
type Options struct {
	Bin      string        // claude 실행 파일(기본 "claude" — PATH에서 해석)
	Model    string        // 선택: --model
	Timeout  time.Duration // 1회 호출 타임아웃(기본 5분)
	Retries  int           // 실패 재시도 횟수(기본 2)
	ExtraEnv []string      // 추가 환경변수(KEY=VAL)
}

// Client — claude 실행기.
type Client struct{ opt Options }

// New — 기본값 채운 Client.
func New(o Options) *Client {
	if o.Bin == "" {
		o.Bin = "claude"
	}
	if o.Timeout == 0 {
		o.Timeout = 5 * time.Minute
	}
	if o.Retries == 0 {
		o.Retries = 2
	}
	return &Client{opt: o}
}

// Run — system + user 프롬프트로 claude를 1회 실행하고 stdout(트림)을 반환.
// system/user는 구분자로 합쳐 stdin으로 전달한다(플래그 버전 의존 제거 + 긴 입력 안전).
func (c *Client) Run(ctx context.Context, system, user string) (string, error) {
	prompt := user
	if strings.TrimSpace(system) != "" {
		prompt = system + "\n\n===== 작업 =====\n\n" + user
	}
	var lastErr error
	for attempt := 0; attempt <= c.opt.Retries; attempt++ {
		if attempt > 0 {
			back := time.Duration(1<<uint(attempt-1)) * 2 * time.Second // 2s,4s,8s...
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(back):
			}
		}
		out, err := c.runOnce(ctx, prompt)
		if err == nil {
			return strings.TrimSpace(out), nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("claude 실행 %d회 모두 실패: %w", c.opt.Retries+1, lastErr)
}

func (c *Client) runOnce(parent context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, c.opt.Timeout)
	defer cancel()

	args := []string{"-p"} // print 모드(비대화). 프롬프트는 stdin.
	if c.opt.Model != "" {
		args = append(args, "--model", c.opt.Model)
	}
	cmd := exec.CommandContext(ctx, c.opt.Bin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = injectPATH(append(os.Environ(), c.opt.ExtraEnv...))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("타임아웃(%s)", c.opt.Timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return "", fmt.Errorf("%v: %s", err, msg)
	}
	return stdout.String(), nil
}

// injectPATH — npm 전역 bin(%APPDATA%\npm)을 PATH 맨 앞에 추가해 claude.cmd를 항상 찾게 한다.
// 무인 환경(부팅 스타트업 등)에서 PATH가 비어 claude를 못 찾던 사고의 영구 방지책.
func injectPATH(env []string) []string {
	var npmBin string
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			npmBin = filepath.Join(appdata, "npm")
		}
	}
	if npmBin == "" {
		return env
	}
	out := make([]string, 0, len(env))
	patched := false
	for _, kv := range env {
		if len(kv) >= 5 && strings.EqualFold(kv[:5], "PATH=") {
			out = append(out, "PATH="+npmBin+string(os.PathListSeparator)+kv[5:])
			patched = true
		} else {
			out = append(out, kv)
		}
	}
	if !patched {
		out = append(out, "PATH="+npmBin)
	}
	return out
}
