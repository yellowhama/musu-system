package pr

import (
	"strings"
	"testing"

	"github.com/yellowhama/musu-system/marketer/internal/seo"
)

func TestOutletLookup(t *testing.T) {
	o, ok := OutletBySlug("nongmin")
	if !ok || o.Name != "농민신문" {
		t.Fatalf("nongmin lookup = %+v, ok=%v", o, ok)
	}
	if _, ok := OutletBySlug("nope"); ok {
		t.Errorf("unknown outlet must return ok=false")
	}
	if len(OutletSlugs()) != len(DefaultOutlets) {
		t.Errorf("OutletSlugs len mismatch")
	}
}

func TestPitchSlug(t *testing.T) {
	p := Pitch{Keyword: "농지법 처분명령", Outlet: Outlet{Slug: "nongmin"}}
	if got := p.Slug(); got != "농지법-처분명령-nongmin" {
		t.Errorf("Slug = %q", got)
	}
}

func TestPitchRender(t *testing.T) {
	pack := seo.NewSourcePack("농지법 처분명령", nil)
	p := Pitch{
		Outlet:  Outlet{Slug: "nongmin", Name: "농민신문", Focus: "농업 정책"},
		Keyword: "농지법 처분명령",
		Subject: "2026 농지 전수조사, 무엇이 달라지나",
		Angle:   "전수조사 시행으로 농지주 관심 급증.",
		Body:    "처분명령은 농지법 제10조에 근거한다 [S1].",
		Pack:    pack,
	}
	out := p.Render()
	for _, want := range []string{
		"status: draft",
		"human-in-the-loop",
		"발송 안 함",
		"농민신문",
		"기사 가치(앵글)",
		"## 근거 출처",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n---\n%s", want, out)
		}
	}
}
