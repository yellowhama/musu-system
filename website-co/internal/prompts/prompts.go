// Package prompts — 작가/편집장 시스템 프롬프트(paperclip AGENTS.md에서 stateless 모델로 이식).
// go:embed로 .md를 바이너리에 박고, {{BRAND}}/{{SLUG}} 등 토큰을 테넌트 값으로 치환한다.
package prompts

import (
	_ "embed"
	"strings"
)

//go:embed writer.md
var writerMD string

//go:embed editor.md
var editorMD string

// Vars — 프롬프트 치환 토큰.
type Vars struct {
	Brand string // {{BRAND}}
	Slug  string // {{SLUG}}
}

func (v Vars) apply(s string) string {
	r := strings.NewReplacer(
		"{{BRAND}}", def(v.Brand, "농지다"),
		"{{SLUG}}", def(v.Slug, "<slug>"),
	)
	return r.Replace(s)
}

// Writer — 치환된 작가 시스템 프롬프트.
func Writer(v Vars) string { return v.apply(writerMD) }

// Editor — 치환된 편집장 시스템 프롬프트.
func Editor(v Vars) string { return v.apply(editorMD) }

func def(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
