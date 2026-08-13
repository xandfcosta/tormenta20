package api

import (
	"context"
	"net/http"
)

type ctxKey int

const userCtxKey ctxKey = iota

// requireAuth verifies the session (cookie or Bearer), loads the user, and stores
// the AuthUser in context. A stale token whose user was deleted is rejected —
// see TestRequireAuthRejectsMissingAndBrokenCredentials.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.extractToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		sub, err := s.verifyToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		user, err := s.queries.GetUserByID(r.Context(), sub)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "User no longer exists")
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, authUserFrom(user))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken reads the JWT from the session cookie, falling back to the
// Authorization: Bearer header, in that order.
func (s *Server) extractToken(r *http.Request) string {
	if c, err := r.Cookie(s.cfg.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return extractBearer(r.Header.Get("Authorization"))
}

// currentUser returns the AuthUser attached by requireAuth (zero value if absent).
func currentUser(r *http.Request) AuthUser {
	u, _ := r.Context().Value(userCtxKey).(AuthUser)
	return u
}
