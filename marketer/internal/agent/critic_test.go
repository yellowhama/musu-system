package agent

import (
	"strings"
	"testing"
)

func TestParseCriticEvaluation(t *testing.T) {
	tests := []struct {
		name                 string
		in                   string
		wantErr              bool
		wantApproved         bool
		wantFeedbackContains string
	}{
		{
			name:                 "approved draft",
			in:                   `{"approved": true, "feedback": "world-class hook, ships as-is"}`,
			wantApproved:         true,
			wantFeedbackContains: "hook",
		},
		{
			name:                 "rejected draft",
			in:                   `{"approved": false, "feedback": "generic AI fluff in the opening line"}`,
			wantApproved:         false,
			wantFeedbackContains: "fluff",
		},
		{
			name:         "approved with empty feedback",
			in:           `{"approved": true, "feedback": ""}`,
			wantApproved: true,
		},
		{
			name:    "truncated json errors",
			in:      `{"approved": true, "feedback":`,
			wantErr: true,
		},
		{
			name:    "empty response errors",
			in:      ``,
			wantErr: true,
		},
		{
			name:    "non-object payload errors",
			in:      `approved`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval, err := parseCriticEvaluation(tt.in)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil (eval=%+v)", eval)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if eval.Approved != tt.wantApproved {
				t.Errorf("approved = %v, want %v", eval.Approved, tt.wantApproved)
			}
			if tt.wantFeedbackContains != "" && !strings.Contains(eval.Feedback, tt.wantFeedbackContains) {
				t.Errorf("feedback %q does not contain %q", eval.Feedback, tt.wantFeedbackContains)
			}
		})
	}
}
