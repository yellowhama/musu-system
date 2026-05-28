package policy

import "testing"

func TestDecide(t *testing.T) {
	c := DefaultConfig()

	tests := []struct {
		name     string
		in       TriageResult
		wantAuto bool
		wantEsc  bool
	}{
		{"refund always escalates even if confident", TriageResult{Category: CategoryRefund, Confidence: 0.99}, false, true},
		{"complaint escalates", TriageResult{Category: CategoryComplaint, Confidence: 0.95}, false, true},
		{"legal escalates", TriageResult{Category: CategoryLegal, Confidence: 0.95}, false, true},
		{"sensitive flag escalates", TriageResult{Category: CategoryFAQ, Confidence: 0.99, Sensitive: true}, false, true},
		{"confident allow-listed faq auto-sends", TriageResult{Category: CategoryFAQ, Confidence: 0.90}, true, false},
		{"confident hours auto-sends", TriageResult{Category: CategoryHours, Confidence: 0.86}, true, false},
		{"allow-listed but below threshold drafts", TriageResult{Category: CategoryFAQ, Confidence: 0.50}, false, false},
		{"confident but not allow-listed (sales) drafts", TriageResult{Category: CategorySales, Confidence: 0.99}, false, false},
		{"other category drafts", TriageResult{Category: CategoryOther, Confidence: 0.99}, false, false},
		{"exactly at threshold auto-sends", TriageResult{Category: CategoryStatus, Confidence: 0.85}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := c.Decide(tt.in)
			if d.AutoSend != tt.wantAuto {
				t.Errorf("AutoSend = %v, want %v (reason: %s)", d.AutoSend, tt.wantAuto, d.Reason)
			}
			if d.Escalate != tt.wantEsc {
				t.Errorf("Escalate = %v, want %v (reason: %s)", d.Escalate, tt.wantEsc, d.Reason)
			}
			if d.AutoSend && d.Escalate {
				t.Errorf("decision is both AutoSend and Escalate")
			}
		})
	}
}
