package publish

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yellowhama/musu-system/website-co/internal/tenant"
)

// Result — 발행 결과.
type Result struct {
	Slug    string
	URL     string
	Skipped bool // 이미 발행됨(멱등)
	Path    string
}

// Publish — 승인 초안을 테넌트 어댑터로 발행하고 원장에 기록한다(write-ahead intent).
// 1) frontmatter 파싱 2) 멱등 체크 3) 드래프트 저장 4) intent 5) 어댑터 발행 6) published 확정.
func Publish(ctx context.Context, cfg tenant.Config, draft string, ledger *Ledger, now time.Time) (Result, error) {
	meta, err := ParseFrontmatter(draft)
	if err != nil {
		return Result{}, err
	}
	if ledger.IsPublished(meta.Slug) {
		return Result{Slug: meta.Slug, Skipped: true}, nil
	}

	// 드래프트 저장(작가가 아닌 Go가 파일 소유)
	if err := os.MkdirAll(cfg.DraftsDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("draftsDir 생성: %w", err)
	}
	draftPath := filepath.Join(cfg.DraftsDir, meta.Slug+".md")
	if err := os.WriteFile(draftPath, []byte(draft), 0o644); err != nil {
		return Result{}, fmt.Errorf("드래프트 저장: %w", err)
	}

	// write-ahead intent
	if err := ledger.MarkIntent(meta.Slug, meta.Title, now); err != nil {
		return Result{}, fmt.Errorf("intent 기록: %w", err)
	}

	url, err := applyAdapter(ctx, cfg.Publish, meta, draftPath)
	if err != nil {
		return Result{}, fmt.Errorf("발행 어댑터(%s): %w", cfg.Publish.Mode, err)
	}

	if err := ledger.MarkPublished(meta.Slug, meta.Title, url, now); err != nil {
		return Result{}, fmt.Errorf("published 확정: %w", err)
	}
	return Result{Slug: meta.Slug, URL: url, Path: draftPath}, nil
}

func applyAdapter(ctx context.Context, p tenant.Publish, meta Meta, draftPath string) (string, error) {
	switch p.Mode {
	case "file":
		if err := os.MkdirAll(p.Dir, 0o755); err != nil {
			return "", err
		}
		dst := filepath.Join(p.Dir, meta.Slug+".md")
		b, err := os.ReadFile(draftPath)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return "", err
		}
		return dst, nil
	case "command":
		args := make([]string, len(p.Argv))
		repl := strings.NewReplacer(
			"{slug}", meta.Slug, "{title}", meta.Title,
			"{category}", meta.Category, "{file}", draftPath,
		)
		for i, a := range p.Argv {
			args[i] = repl.Replace(a)
		}
		if len(args) == 0 {
			return "", fmt.Errorf("argv 비어있음")
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...) // 셸 없이 execFile(인젝션 방지)
		if p.Cwd != "" {
			cmd.Dir = p.Cwd
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			msg := out.String()
			if len(msg) > 400 {
				msg = msg[:400]
			}
			return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(msg))
		}
		return extractURL(out.String()), nil
	default:
		return "", fmt.Errorf("알 수 없는 mode: %q", p.Mode)
	}
}

// extractURL — 발행 명령 출력에서 https URL 1개를 뽑는다(있으면).
func extractURL(s string) string {
	for _, tok := range strings.Fields(s) {
		if strings.HasPrefix(tok, "https://") {
			return strings.TrimRight(tok, ".,)\"'")
		}
	}
	return ""
}
