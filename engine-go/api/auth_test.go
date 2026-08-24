package api

import (
	"t20engine/plataforma"
	"testing"
	"time"

	"t20engine/db/sqlcgen"
)

func testServer(secret string) *Server {
	return &Server{cfg: plataforma.Config{JWTSecret: secret, JWTExpiresIn: "7d", CookieName: "t20_session"}}
}

func TestSignVerifyRoundtrip(t *testing.T) {
	s := testServer("secret-key")
	tok, err := s.signToken(sqlcgen.User{ID: 42, Email: "a@b.com"})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sub, err := s.verifyToken(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if sub != 42 {
		t.Fatalf("sub = %d, want 42", sub)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	tok, _ := testServer("real").signToken(sqlcgen.User{ID: 1, Email: "x@y.com"})
	if _, err := testServer("forged").verifyToken(tok); err == nil {
		t.Fatal("verify accepted a token signed with a different secret")
	}
}

func TestParseExpiry(t *testing.T) {
	cases := map[string]time.Duration{
		"7d":  7 * 24 * time.Hour,
		"12h": 12 * time.Hour,
		"30m": 30 * time.Minute,
		"":    sessionTTL,
		"abc": sessionTTL,
	}
	for in, want := range cases {
		if got := parseExpiry(in); got != want {
			t.Errorf("parseExpiry(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestValidateRegister(t *testing.T) {
	if f := ValidateRegister(registerBody{Email: "gm@test.com", Password: "password123"}); len(f) != 0 {
		t.Fatalf("valid input flagged: %v", f)
	}
	f := ValidateRegister(registerBody{Email: "bad", Password: "short"})
	if _, ok := f["email"]; !ok {
		t.Error("bad email not flagged")
	}
	if _, ok := f["password"]; !ok {
		t.Error("short password not flagged")
	}
}
