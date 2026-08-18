package fabric

import (
	"bytes"
	"testing"
	"time"
)

func TestStaticAndShortLivedCredentials(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	user := "AbC123"

	static := StaticCredential(key, user)
	if !VerifyCredential(key, "abc123", static, time.Now()) {
		t.Fatal("static credential must verify for normalized user")
	}
	if VerifyCredential(key, "other", static, time.Now()) {
		t.Fatal("static credential must not verify for another user")
	}

	token, err := IssueToken(key, user, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyCredential(key, user, token, time.Now()) {
		t.Fatal("short-lived token must verify")
	}
	if VerifyCredential(key, user, token, time.Now().Add(10*time.Minute)) {
		t.Fatal("expired short-lived token must fail")
	}
}
