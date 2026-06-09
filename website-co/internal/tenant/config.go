// Package tenant — 멀티테넌트 config 로더. 사이트(농지다·vibecode.town) = 유저 데이터(config 1개).
// 운영 로직은 하나, 사이트는 데이터. tenants/<site>/config.json.
package tenant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Publish — 발행 어댑터 설정. mode=file(드래프트를 dir에 복사) 또는 command(외부 명령, {{...}} 치환).
type Publish struct {
	Mode string   `json:"mode"`           // "file" | "command"
	Dir  string   `json:"dir,omitempty"`  // file 모드: 발행 대상 디렉토리
	Argv []string `json:"argv,omitempty"` // command 모드: 실행 인자(토큰 치환). 셸 없이 execFile.
	Cwd  string   `json:"cwd,omitempty"`  // command 모드: 작업 디렉토리(예: njd wiki-gen은 api/에서)
}

// Config — 한 테넌트(사이트)의 운영 설정.
type Config struct {
	Site       string   `json:"site"`        // 테넌트 식별자(디렉토리명)
	Brand      string   `json:"brand"`       // {{BRAND}} — 작가/편집장 프롬프트 치환
	Niche      string   `json:"niche"`       // 토픽 발굴용(Phase 3)
	URL        string   `json:"url"`         // 사이트 URL
	DraftsDir  string   `json:"draftsDir"`   // 작가 초안 저장 경로
	LedgerPath string   `json:"ledgerPath"`  // 발행원장 경로(기존 published-approved.json 공유 가능 → 중복방지)
	Publish    Publish  `json:"publish"`     // 발행 어댑터
	MaxRounds  int      `json:"maxRounds"`   // 수정 루프 캡(기본 4)
	WriterCWDs []string `json:"writerCwds"`  // (선택) 작가 작업 디렉토리 회전 — stateless라 보통 불요
	Schedule   string   `json:"schedule"`    // (Phase 3) cron
	SecretsRef string   `json:"secretsRef"`  // (선택) 시크릿 파일 참조 경로(값 비노출)
}

// Load — tenants/<site>/config.json 1개를 읽는다.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("테넌트 config 읽기(%s): %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("테넌트 config 파싱(%s): %w", path, err)
	}
	if c.Site == "" {
		c.Site = strings.TrimSuffix(filepath.Base(filepath.Dir(path)), "")
	}
	if c.MaxRounds <= 0 {
		c.MaxRounds = 4
	}
	if err := c.validate(); err != nil {
		return Config{}, fmt.Errorf("테넌트 config 검증(%s): %w", c.Site, err)
	}
	return c, nil
}

// LoadAll — tenants/ 하위 각 디렉토리의 config.json을 모두 로드(멀티테넌트).
func LoadAll(tenantsDir string) ([]Config, error) {
	entries, err := os.ReadDir(tenantsDir)
	if err != nil {
		return nil, fmt.Errorf("tenants 디렉토리(%s): %w", tenantsDir, err)
	}
	var out []Config
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(tenantsDir, e.Name(), "config.json")
		if _, err := os.Stat(p); err != nil {
			continue // config.json 없는 디렉토리는 스킵
		}
		c, err := Load(p)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Brand) == "" {
		return fmt.Errorf("brand 필수")
	}
	if strings.TrimSpace(c.DraftsDir) == "" {
		return fmt.Errorf("draftsDir 필수")
	}
	switch c.Publish.Mode {
	case "file":
		if c.Publish.Dir == "" {
			return fmt.Errorf("publish.mode=file 인데 publish.dir 없음")
		}
	case "command":
		if len(c.Publish.Argv) == 0 {
			return fmt.Errorf("publish.mode=command 인데 publish.argv 없음")
		}
	case "":
		return fmt.Errorf("publish.mode 필수(file|command)")
	default:
		return fmt.Errorf("publish.mode 알 수 없음: %q", c.Publish.Mode)
	}
	return nil
}
