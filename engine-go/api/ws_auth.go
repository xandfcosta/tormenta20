package api

import (
	"context"
	"errors"
	"net/http"
)

// authenticateHandshake authenticates a socket handshake, mirroring ws-auth.ts: resolve
// the JWT from handshake auth.token → Authorization: Bearer → the session cookie (that
// exact order), verify it with the same secret as the HTTP layer, and load the user so a
// stale token whose user was deleted is rejected. It takes the raw handshake inputs — not
// a socket — so the auth rule stays transport-agnostic over the WS library and is unit
// testable without a live connection. The gateway glue extracts these three from the
// zishang520 socket (Handshake().Auth["token"], the Authorization + Cookie headers).
func (s *Server) authenticateHandshake(ctx context.Context, authToken, authHeader, cookieHeader string) (AuthUser, error) {
	token := handshakeToken(authToken, authHeader, cookieHeader, s.cfg.CookieName)
	if token == "" {
		return AuthUser{}, errors.New("Missing auth token")
	}
	sub, err := s.verifyToken(token)
	if err != nil {
		return AuthUser{}, errors.New("Invalid auth token")
	}
	user, err := s.queries.GetUserByID(ctx, sub)
	if err != nil {
		return AuthUser{}, errors.New("User no longer exists")
	}
	return authUserFrom(user), nil
}

// handshakeToken picks the JWT from the handshake sources in ws-auth.ts order: an explicit
// auth.token, then Authorization: Bearer, then the session cookie.
func handshakeToken(authToken, authHeader, cookieHeader, cookieName string) string {
	if authToken != "" {
		return authToken
	}
	if b := extractBearer(authHeader); b != "" {
		return b
	}
	return cookieToken(cookieHeader, cookieName)
}

// cookieToken parses a single cookie value out of a raw Cookie header, reusing the stdlib
// cookie grammar (a synthetic request avoids re-implementing it) — same read the HTTP
// middleware does via r.Cookie, so cookies issued by issueSession resolve identically.
func cookieToken(cookieHeader, name string) string {
	if cookieHeader == "" {
		return ""
	}
	r := &http.Request{Header: http.Header{"Cookie": []string{cookieHeader}}}
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}
