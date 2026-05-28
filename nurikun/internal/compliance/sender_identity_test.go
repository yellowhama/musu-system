package compliance

import (
	"strings"
	"testing"
)

func TestValidateSenderIdentity(t *testing.T) {
	cases := []struct {
		name     string
		sender   string
		postal   string
		wantErr  bool
		wantSubs string // substring that must appear in the error message
	}{
		{"both real", "농지다", "경기도 어딘가 1-1", false, ""},
		{"empty sender", "", "경기도 어딘가 1-1", true, "sender_name is empty"},
		{"empty postal", "농지다", "", true, "sender_physical is empty"},
		{"placeholder ascii", "농지다", "PLACEHOLDER_실주소_미설정_운영자_수정필요", true, "PLACEHOLDER"},
		{"placeholder mixed case", "농지다", "placeholder address", true, "placeholder"},
		{"todo in sender name", "농지다 TODO", "Seoul", true, "TODO"},
		{"example.com leak", "농지다", "ops@example.com", true, "example.com"},
		{"미설정 한글 패턴", "농지다", "주소 미설정", true, "미설정"},
		{"수정필요 한글 패턴", "농지다", "운영자 수정필요", true, "수정필요"},
		{"both valid Korean", "농지다", "서울특별시 종로구 광화문로 1", false, ""},
		{"whitespace only sender", "   ", "Seoul", true, "sender_name is empty"},
		{"whitespace only postal", "농지다", "   ", true, "sender_physical is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSenderIdentity(tc.sender, tc.postal)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantSubs != "" && !strings.Contains(err.Error(), tc.wantSubs) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSubs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWarnPublicBaseURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{"empty", "", ""},
		{"clean https", "https://nurikun.nongjida.kr", ""},
		{"http public", "http://nurikun.nongjida.kr", ""},
		{"localhost", "http://localhost:8088", "localhost"},
		{"127.0.0.1", "http://127.0.0.1:8088", "127.0.0.1"},
		{"ipv6 loopback", "http://[::1]:8088", "::1"},
		{"bind any", "http://0.0.0.0:8088", "0.0.0.0"},
		{"mixed case localhost", "https://LOCALHOST/x", "localhost"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WarnPublicBaseURL(tc.url)
			if tc.want == "" {
				if got != "" {
					t.Fatalf("expected no warning, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected warning to contain %q, got %q", tc.want, got)
			}
		})
	}
}
