package agent

import "testing"

func TestValidateDraft(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"normal draft", "Hook line.\nBody copy.", false},
		{"single char", "x", false},
		{"content with surrounding space is kept verbatim", "  real content  ", false},
		{"empty string is rejected", "", true},
		{"whitespace only is rejected", "   \n\t  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateDraft(tt.in)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.in {
				t.Errorf("validateDraft(%q) = %q, want the input returned unchanged", tt.in, got)
			}
		})
	}
}
