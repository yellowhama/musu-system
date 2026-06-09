// Package publish — 신뢰 발행기. 편집장 승인 초안을 테넌트 어댑터(file/command)로 발행하고 원장에 기록.
// write-ahead intent로 중복발행 방지. claude는 관여 안 함(Go가 발행 소유).
package publish

import (
	"fmt"
	"regexp"
	"strings"
)

// Meta — 초안 frontmatter에서 뽑은 발행 메타.
type Meta struct {
	Slug     string
	Title    string
	Category string
}

var fmKeyRe = regexp.MustCompile(`(?m)^(title|slug|category):\s*(.+?)\s*$`)

// ParseFrontmatter — 초안 상단 frontmatter에서 title/slug/category를 추출한다.
// `---` 구분자 유무 모두 허용(상단 라인 스캔). 값의 따옴표는 벗긴다.
func ParseFrontmatter(draft string) (Meta, error) {
	head := draft
	// `---` 펜스가 있으면 그 안만 본다.
	if strings.HasPrefix(strings.TrimSpace(draft), "---") {
		t := strings.TrimSpace(draft)
		if i := strings.Index(t[3:], "---"); i >= 0 {
			head = t[3 : 3+i]
		}
	} else if i := strings.Index(draft, "\n## "); i > 0 {
		head = draft[:i] // 첫 본문 섹션 전까지
	}
	var m Meta
	for _, mm := range fmKeyRe.FindAllStringSubmatch(head, -1) {
		v := strings.Trim(strings.TrimSpace(mm[2]), `"'`)
		switch mm[1] {
		case "title":
			m.Title = v
		case "slug":
			m.Slug = v
		case "category":
			m.Category = v
		}
	}
	if m.Slug == "" || m.Title == "" {
		return Meta{}, fmt.Errorf("frontmatter에 slug/title 누락 (slug=%q title=%q)", m.Slug, m.Title)
	}
	if m.Category == "" {
		m.Category = "News"
	}
	return m, nil
}
