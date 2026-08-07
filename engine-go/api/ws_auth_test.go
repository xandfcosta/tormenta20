package api

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// mintToken signs a JWT with the given sub using the server's secret, matching what
// issueSession produces (sub as a JSON number).
func mintToken(t *testing.T, secret string, sub int64) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": float64(sub)})
	str, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return str
}

func TestAuthenticateHandshake(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	uid := seedUser(t, s, "sock@t.com")
	valid := mintToken(t, s.cfg.JWTSecret, uid)

	t.Run("via auth.token", func(t *testing.T) {
		user, err := s.authenticateHandshake(ctx, valid, "", "")
		if err != nil || user.ID != uid {
			t.Fatalf("user=%+v err=%v", user, err)
		}
	})
	t.Run("via Authorization Bearer", func(t *testing.T) {
		if user, err := s.authenticateHandshake(ctx, "", "Bearer "+valid, ""); err != nil || user.ID != uid {
			t.Fatalf("user=%+v err=%v", user, err)
		}
	})
	t.Run("via session cookie", func(t *testing.T) {
		if user, err := s.authenticateHandshake(ctx, "", "", "t20_session="+valid); err != nil || user.ID != uid {
			t.Fatalf("user=%+v err=%v", user, err)
		}
	})
	t.Run("auth.token wins over cookie (order)", func(t *testing.T) {
		other := mintToken(t, s.cfg.JWTSecret, 999999)
		user, err := s.authenticateHandshake(ctx, valid, "", "t20_session="+other)
		if err != nil || user.ID != uid {
			t.Fatalf("auth.token should take precedence: user=%+v err=%v", user, err)
		}
	})
	t.Run("no token → missing", func(t *testing.T) {
		if _, err := s.authenticateHandshake(ctx, "", "", ""); err == nil || err.Error() != "Missing auth token" {
			t.Errorf("err=%v, want 'Missing auth token'", err)
		}
	})
	t.Run("garbage token → invalid", func(t *testing.T) {
		if _, err := s.authenticateHandshake(ctx, "not-a-jwt", "", ""); err == nil || err.Error() != "Invalid auth token" {
			t.Errorf("err=%v, want 'Invalid auth token'", err)
		}
	})
	t.Run("valid signature but deleted user → rejected", func(t *testing.T) {
		ghost := mintToken(t, s.cfg.JWTSecret, 424242)
		if _, err := s.authenticateHandshake(ctx, ghost, "", ""); err == nil || err.Error() != "User no longer exists" {
			t.Errorf("err=%v, want 'User no longer exists'", err)
		}
	})
	t.Run("token signed with wrong secret → invalid", func(t *testing.T) {
		bad := mintToken(t, "wrong-secret", uid)
		if _, err := s.authenticateHandshake(ctx, bad, "", ""); err == nil || err.Error() != "Invalid auth token" {
			t.Errorf("err=%v, want 'Invalid auth token'", err)
		}
	})
}
