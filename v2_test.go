package main

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestValidV2Envelope(t *testing.T) {
	valid := `{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`
	if !validV2Envelope(valid, len(valid)) {
		t.Fatal("expected valid envelope")
	}

	invalid := []string{
		`{"v":1,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`,
		`{"v":2,"n":"short","c":"AAAAAAAAAAAAAAAAAAAAAA"}`,
		`{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"short"}`,
		valid + "x",
	}
	for _, value := range invalid {
		if validV2Envelope(value, len(valid)) {
			t.Fatalf("accepted invalid envelope: %s", value)
		}
	}
}

func TestLookupTokenEncoding(t *testing.T) {
	token := make([]byte, 32)
	encoded := base64.RawURLEncoding.EncodeToString(token)
	if encoded != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" {
		t.Fatalf("unexpected token encoding: %s", encoded)
	}
	if got := sha256Hex(token); got != "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925" {
		t.Fatalf("unexpected lookup hash: %s", got)
	}
}

func TestFixedWindowLimiter(t *testing.T) {
	limiter := newFixedWindowLimiter(2, time.Minute)
	if !limiter.allow("client") || !limiter.allow("client") {
		t.Fatal("expected requests within limit to pass")
	}
	if limiter.allow("client") {
		t.Fatal("expected request over limit to fail")
	}
}
