package seo

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"농지법 처분명령 대응":        "농지법-처분명령-대응",
		"What is 농지법?":         "what-is-농지법",
		"  trailing  spaces  ": "trailing-spaces",
		"a---b":                "a-b",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestArticleRender(t *testing.T) {
	pack := testPack()
	art := Article{
		Outline: Outline{
			Title:             "농지법 처분명령, 무엇을 해야 하나",
			MetaDescription:   "처분명령을 받았을 때의 대응 절차를 정리합니다.",
			Slug:              "nongjibeop-cheobun",
			TargetKeyword:     "농지법 처분명령",
			SecondaryKeywords: []string{"이행강제금", "농지 처분"},
		},
		Body: "농지법 제10조에 따라 처분의무가 발생한다 [S1].",
		Pack: pack,
	}
	out := art.Render()

	for _, must := range []string{
		"---\n",
		`title: "농지법 처분명령, 무엇을 해야 하나"`,
		`slug: "nongjibeop-cheobun"`,
		"secondary_keywords:",
		"lang: ko-KR",
		`"@type": "Article"`,
		"application/ld+json",
		"## 출처",
		"**[S1]** 농지법 제10조 — 법제처",
		"**[S2]** 2026 농지 전수조사 — 농림축산식품부",
	} {
		if !strings.Contains(out, must) {
			t.Errorf("rendered article missing %q\n---\n%s", must, out)
		}
	}
}

func TestSourcePackPrompt(t *testing.T) {
	p := testPack()
	got := p.Prompt()
	if !strings.Contains(got, "[S1] 농지법 제10조 (출처: 법제처)") {
		t.Errorf("prompt missing labeled source:\n%s", got)
	}
	if !strings.Contains(got, "[S2] 2026 농지 전수조사 (출처: 농림축산식품부)") {
		t.Errorf("prompt missing S2:\n%s", got)
	}
}
