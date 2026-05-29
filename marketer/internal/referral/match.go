// Package referral implements the Sprint 5 lawyer-referral matcher — the BOFU
// step the funnel plan promises. It is pure and deterministic (no LLM, no
// network) and operates on EXPORTED roster files, never on live nurikun
// subscriber data: the operator exports consenting high-intent leads to a JSON
// file, and the matcher proposes ranked lead↔lawyer connections as a DRAFT for
// human review. Nothing is connected or sent automatically — outbound stays
// human-in-the-loop, and matching consented leads to partner lawyers is squarely
// on the healthy (legal, not-cringe) side of the line.
package referral

import (
	"fmt"
	"sort"
	"strings"
)

// Lead is a consenting, high-intent prospect exported for referral. Region and
// Situation drive matching; Urgency (1-5) prioritizes scarce lawyer capacity.
type Lead struct {
	ID        string `json:"id"`
	Region    string `json:"region"`    // e.g. "전북", "경기"
	Situation string `json:"situation"` // e.g. "처분명령", "상속", "귀농", "매입"
	Urgency   int    `json:"urgency"`   // 1..5
	Note      string `json:"note,omitempty"`
}

// Lawyer is a partner attorney with coverage, specialties, and remaining
// capacity for new referrals this cycle.
type Lawyer struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Regions     []string `json:"regions"`     // "전국" matches any region
	Specialties []string `json:"specialties"` // matched against Lead.Situation
	Capacity    int      `json:"capacity"`    // remaining referral slots
}

func (l Lawyer) coversRegion(region string) bool {
	for _, r := range l.Regions {
		if r == "전국" || r == region {
			return true
		}
	}
	return false
}

func (l Lawyer) hasSpecialty(situation string) bool {
	for _, s := range l.Specialties {
		if s == situation {
			return true
		}
	}
	return false
}

// Match is one proposed connection with a transparent score and reasons.
type Match struct {
	Lead    Lead
	Lawyer  *Lawyer // nil when no lawyer with capacity fit
	Score   int
	Reasons []string
}

// score rates a lawyer for a lead. Specialty relevance is REQUIRED: matching a
// lead to a lawyer who doesn't handle their situation is a low-quality (cringe)
// referral, so a specialty miss means not eligible. Region is a strong booster
// (prefer a local specialist), and urgency is a tiebreaker so scarce capacity
// goes to the most pressing leads. Returns (score, reasons, eligible).
func score(lead Lead, law Lawyer) (int, []string, bool) {
	if !law.hasSpecialty(lead.Situation) {
		return 0, nil, false
	}
	s := 40
	reasons := []string{"전문분야 일치(" + lead.Situation + ")"}
	if law.coversRegion(lead.Region) {
		s += 50
		reasons = append(reasons, "지역 일치("+lead.Region+")")
	}
	s += clampUrgency(lead.Urgency) * 2
	return s, reasons, true
}

func clampUrgency(u int) int {
	if u < 1 {
		return 1
	}
	if u > 5 {
		return 5
	}
	return u
}

// MatchLeads greedily assigns each lead its best eligible lawyer with remaining
// capacity. Leads are processed most-urgent-first (stable by ID for ties) so the
// result is deterministic, and capacity is decremented as it is consumed. Leads
// with no eligible lawyer come back with Lawyer == nil for the operator to see.
func MatchLeads(leads []Lead, lawyers []Lawyer) []Match {
	// Work on a capacity copy so the caller's roster is not mutated.
	cap := make([]int, len(lawyers))
	for i, l := range lawyers {
		cap[i] = l.Capacity
	}

	order := make([]int, len(leads))
	for i := range leads {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		la, lb := leads[order[a]], leads[order[b]]
		if clampUrgency(la.Urgency) != clampUrgency(lb.Urgency) {
			return clampUrgency(la.Urgency) > clampUrgency(lb.Urgency)
		}
		return la.ID < lb.ID
	})

	out := make([]Match, len(leads))
	for _, idx := range order {
		lead := leads[idx]
		best, bestScore, bestReasons := -1, -1, []string(nil)
		for j, law := range lawyers {
			if cap[j] <= 0 {
				continue
			}
			sc, reasons, ok := score(lead, law)
			if !ok {
				continue
			}
			// Deterministic tie-break: higher score, then earlier roster index.
			if sc > bestScore {
				best, bestScore, bestReasons = j, sc, reasons
			}
		}
		m := Match{Lead: lead}
		if best >= 0 {
			cap[best]--
			law := lawyers[best]
			m.Lawyer = &law
			m.Score = bestScore
			m.Reasons = bestReasons
		}
		out[idx] = m // keep input order in the result
	}
	return out
}

// Render produces the reviewable referral draft. It makes the human-in-the-loop
// contract explicit and lists unmatched leads separately so none are silently
// dropped.
func Render(matches []Match) string {
	var b strings.Builder
	b.WriteString("---\nkind: referral-draft\n")
	b.WriteString("delivery: human-in-the-loop (운영자가 검토 후 직접 연결)\n---\n\n")
	b.WriteString("# 변호사 referral 매칭 초안\n\n")
	b.WriteString("> ⚠️ 초안입니다. 동의한 리드만 포함되어야 하며, 실제 연결은 운영자가 검토 후 진행하세요. 자동 연결/발송 없음.\n\n")

	var matched, unmatched []Match
	for _, m := range matches {
		if m.Lawyer != nil {
			matched = append(matched, m)
		} else {
			unmatched = append(unmatched, m)
		}
	}

	b.WriteString(fmt.Sprintf("## 매칭 %d건\n\n", len(matched)))
	if len(matched) > 0 {
		b.WriteString("| Lead | 지역 | 상황 | 긴급도 | 추천 변호사 | 점수 | 근거 |\n|---|---|---|---|---|---|---|\n")
		for _, m := range matched {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %d | %s |\n",
				m.Lead.ID, m.Lead.Region, m.Lead.Situation, clampUrgency(m.Lead.Urgency),
				m.Lawyer.Name, m.Score, strings.Join(m.Reasons, ", ")))
		}
	}

	b.WriteString(fmt.Sprintf("\n## 미매칭 %d건 (수동 검토 필요)\n\n", len(unmatched)))
	for _, m := range unmatched {
		b.WriteString(fmt.Sprintf("- %s (%s / %s) — 적합한 가용 변호사 없음\n",
			m.Lead.ID, m.Lead.Region, m.Lead.Situation))
	}
	return b.String()
}
