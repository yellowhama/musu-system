package compliance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignUnsub returns a hex HMAC-SHA256 of email under secret. It makes one-click
// unsubscribe links unforgeable, so a public endpoint cannot be used to opt out
// arbitrary addresses. An empty secret yields "" (signing disabled).
func SignUnsub(email, secret string) string {
	if secret == "" {
		return ""
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(email))
	return hex.EncodeToString(m.Sum(nil))
}

// VerifyUnsub reports (in constant time) whether sig is a valid signature for
// email under secret. With an empty secret or signature it fails closed.
func VerifyUnsub(email, sig, secret string) bool {
	if secret == "" || sig == "" {
		return false
	}
	return hmac.Equal([]byte(SignUnsub(email, secret)), []byte(sig))
}
