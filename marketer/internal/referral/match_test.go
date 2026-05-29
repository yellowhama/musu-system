package referral

import (
	"strings"
	"testing"
)

func sampleLawyers() []Lawyer {
	return []Lawyer{
		{ID: "L1", Name: "김농지", Regions: []string{"전북"}, Specialties: []string{"처분명령"}, Capacity: 1},
		{ID: "L2", Name: "이상속", Regions: []string{"전국"}, Specialties: []string{"상속"}, Capacity: 2},
		{ID: "L3", Name: "박매입", Regions: []string{"경기"}, Specialties: []string{"매입"}, Capacity: 1},
	}
}

func TestMatchPrefersRegionAndSpecialty(t *testing.T) {
	leads := []Lead{{ID: "A", Region: "전북", Situation: "처분명령", Urgency: 3}}
	m := MatchLeads(leads, sampleLawyers())
	if m[0].Lawyer == nil || m[0].Lawyer.ID != "L1" {
		t.Fatalf("expected L1 (region+specialty), got %+v", m[0].Lawyer)
	}
	// 50 (region) + 40 (specialty) + 3*2 = 96
	if m[0].Score != 96 {
		t.Errorf("score = %d, want 96", m[0].Score)
	}
}

func TestCapacityIsRespectedAndUrgencyOrdered(t *testing.T) {
	// Two 상속 leads, only L2 fits (cap 2). A third 상속 lead would be unmatched.
	leads := []Lead{
		{ID: "low", Region: "강원", Situation: "상속", Urgency: 1},
		{ID: "high", Region: "강원", Situation: "상속", Urgency: 5},
		{ID: "extra", Region: "강원", Situation: "상속", Urgency: 3},
	}
	m := MatchLeads(leads, sampleLawyers())
	byID := map[string]Match{}
	for _, x := range m {
		byID[x.Lead.ID] = x
	}
	// L2 capacity 2 → the two most urgent (high, extra) get it; low is squeezed out.
	if byID["high"].Lawyer == nil || byID["extra"].Lawyer == nil {
		t.Fatalf("urgent leads must be matched: high=%v extra=%v", byID["high"].Lawyer, byID["extra"].Lawyer)
	}
	if byID["low"].Lawyer != nil {
		t.Errorf("least-urgent lead should be unmatched when capacity exhausted")
	}
	// Result preserves input order.
	if m[0].Lead.ID != "low" || m[1].Lead.ID != "high" || m[2].Lead.ID != "extra" {
		t.Errorf("result order not preserved: %s %s %s", m[0].Lead.ID, m[1].Lead.ID, m[2].Lead.ID)
	}
}

func TestNoEligibleLawyerUnmatched(t *testing.T) {
	leads := []Lead{{ID: "Z", Region: "제주", Situation: "임대", Urgency: 4}}
	m := MatchLeads(leads, sampleLawyers())
	if m[0].Lawyer != nil {
		t.Errorf("no region/specialty fit must stay unmatched, got %+v", m[0].Lawyer)
	}
}

func TestRenderSeparatesMatchedAndUnmatched(t *testing.T) {
	leads := []Lead{
		{ID: "A", Region: "전북", Situation: "처분명령", Urgency: 3},
		{ID: "Z", Region: "제주", Situation: "임대", Urgency: 4},
	}
	out := Render(MatchLeads(leads, sampleLawyers()))
	for _, want := range []string{
		"kind: referral-draft",
		"human-in-the-loop",
		"매칭 1건",
		"미매칭 1건",
		"김농지",
		"Z (제주 / 임대)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n---\n%s", want, out)
		}
	}
}
