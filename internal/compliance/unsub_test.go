package compliance

import "testing"

func TestSignVerifyUnsub(t *testing.T) {
	const secret = "s3cr3t"
	email := "a@example.com"

	sig := SignUnsub(email, secret)
	if sig == "" {
		t.Fatal("expected a non-empty signature")
	}
	if !VerifyUnsub(email, sig, secret) {
		t.Error("a valid signature should verify")
	}

	// A signature must not verify for a different email (no opting out others).
	if VerifyUnsub("b@example.com", sig, secret) {
		t.Error("signature must not verify for a different email")
	}
	// Wrong secret must fail.
	if VerifyUnsub(email, sig, "other-secret") {
		t.Error("signature must not verify under a different secret")
	}
	// Empty secret disables signing and fails verification closed.
	if SignUnsub(email, "") != "" {
		t.Error("empty secret should yield an empty signature")
	}
	if VerifyUnsub(email, sig, "") {
		t.Error("empty secret must fail verification")
	}
	if VerifyUnsub(email, "", secret) {
		t.Error("empty signature must fail verification")
	}
}
