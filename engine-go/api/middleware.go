package api

import (
	"context"
	"errors"
	"net/http"
)

type ctxKey int

const userCtxKey ctxKey = iota

// requireAuth verifies the session (cookie or Bearer), loads the user, and stores
// the AuthUser in context. A stale token whose user was deleted is rejected —
// see TestRequireAuthRejectsMissingAndBrokenCredentials.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.sessionUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// errNoSession and errUserGone are separate because the SECOND one is worth
// saying out loud: a token that verifies against a user that no longer exists
// is a stale cookie, not a forged one, and the message is the only place that
// difference is visible.
var (
	errNoSession = errors.New("Unauthorized")
	errUserGone  = errors.New("User no longer exists")
)

// sessionUser resolves the identity a request carries, WITHOUT the middleware.
//
// Extracted for the porta (ALE-229), which has to ask "is anyone logged in?" to
// decide between drawing the login screen and redirecting — a question the
// middleware could only answer by refusing the request. Same shape as
// `deleteAccount` and `authenticate`: the rule was welded to the one transport
// that had needed it.
func (s *Server) sessionUser(r *http.Request) (AuthUser, error) {
	token := s.extractToken(r)
	if token == "" {
		return AuthUser{}, errNoSession
	}
	sub, err := s.verifyToken(token)
	if err != nil {
		return AuthUser{}, errNoSession
	}
	user, err := s.queries.GetUserByID(r.Context(), sub)
	if err != nil {
		return AuthUser{}, errUserGone
	}
	return s.authUser(user), nil
}

// requireAdmin gates the administration routes, and runs AFTER requireAuth —
// it reads the identity that middleware attached. The answer comes from
// ADMIN_EMAILS in the environment file, so the only way to gain it is editing
// that file on the host: no request can grant it (ALE-120).
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).IsAdmin {
			writeError(w, http.StatusForbidden, "Admin only")
			return
		}
		next.ServeHTTP(w, r)
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
