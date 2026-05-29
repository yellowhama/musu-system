package funnel

import (
	"strings"
	"testing"
)

func TestComputeMath(t *testing.T) {
	cfg := DefaultConfig() // 10000, optin 0.03, member 0.5, 150/art, x1.6, 12mo
	m := cfg.Compute()
	// 10000 / 0.5 = 20000 opt-ins; / 0.03 = 666,667 visitors.
	if m.OptInsNeeded != 20000 {
		t.Errorf("OptInsNeeded = %d, want 20000", m.OptInsNeeded)
	}
	if m.VisitorsNeeded != 666667 {
		t.Errorf("VisitorsNeeded = %d, want 666667", m.VisitorsNeeded)
	}
	// 666667 / 12 = 55,556/mo; / (150*1.6=240) = ceil(231.4) = 232 articles.
	if m.MonthlyVisits != 55556 {
		t.Errorf("MonthlyVisits = %d, want 55556", m.MonthlyVisits)
	}
	if m.ArticlesNeeded != 232 {
		t.Errorf("ArticlesNeeded = %d, want 232", m.ArticlesNeeded)
	}
}

func TestComputeGuardsZeroRates(t *testing.T) {
	cfg := Config{TargetMembers: 1000} // all rates zero -> defaults applied
	m := cfg.Compute()
	if m.OptInsNeeded <= 0 || m.VisitorsNeeded <= 0 || m.ArticlesNeeded <= 0 {
		t.Fatalf("zero-rate config must not divide-by-zero: %+v", m)
	}
}

func TestBuildPlanSchedule(t *testing.T) {
	plan := BuildPlan("njd", []string{"농지법 처분명령", "농지 상속"}, DefaultConfig())
	// 3 actions per keyword (2 keywords = 6) + 4 standing actions = 10.
	if len(plan.Actions) != 10 {
		t.Fatalf("Actions = %d, want 10", len(plan.Actions))
	}
	// Second keyword's pillar article is scheduled a week after the first.
	var pillarDays []int
	for _, a := range plan.Actions {
		if a.Tool == "seo" {
			pillarDays = append(pillarDays, a.Day)
		}
	}
	if len(pillarDays) != 2 || pillarDays[0] != 1 || pillarDays[1] != 8 {
		t.Errorf("pillar days = %v, want [1 8]", pillarDays)
	}
}

func TestRenderContainsKeyParts(t *testing.T) {
	plan := BuildPlan("njd", []string{"농지법 처분명령"}, DefaultConfig())
	out := plan.Render()
	for _, want := range []string{
		"kind: acquisition-funnel-plan",
		"필요 구독자(opt-in)",
		"666,667", // comma-formatted visitors
		"캠페인 캘린더",
		"Citation Gate",
		"human-in-the-loop",
		"--all-personas",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func TestCommaInt(t *testing.T) {
	cases := map[int]string{0: "0", 100: "100", 1000: "1,000", 666667: "666,667", -1234: "-1,234"}
	for in, want := range cases {
		if got := commaInt(in); got != want {
			t.Errorf("commaInt(%d) = %q, want %q", in, got, want)
		}
	}
}
