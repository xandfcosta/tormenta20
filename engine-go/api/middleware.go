package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"t20engine/plataforma"
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
			plataforma.WriteError(w, http.StatusUnauthorized, err.Error())
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

// requirePagina é o `requireAuth` das PÁGINAS: quem não tem sessão vai para a
// porta, lembrando para onde ia.
//
// A diferença não é cosmética. O `requireAuth` responde um JSON 401, que é a
// resposta certa para quem chama a API e a errada para quem digitou uma URL: o
// jogador vê `{"statusCode":401}` numa tela branca em vez da tela de entrar.
// Era o guarda `requireSession` da SPA, que morava no cliente (ALE-231).
//
// Vale para TODA página do Datastar, e não só para o Hub — a Mesa e a
// administração tinham a mesma aresta desde o piloto.
func (s *Server) requirePagina(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.sessionUser(r)
		if err != nil {
			http.Redirect(w, r, "/piloto/entrar?redirect="+url.QueryEscape(alvoOriginal(r)), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}

// alvoOriginal é a URL que o navegador pediu, com prefixo e query.
//
// `RequestURI` e não `URL.Path`: o roteador do piloto é montado com
// `http.StripPrefix`, que REESCREVE o caminho — quem lesse `URL.Path` mandaria
// o jogador de volta para `/mesa/1/4` em vez de `/piloto/mesa/1/4`, e ele
// entraria para cair num 404. O `RequestURI` é o alvo cru, intocado pelo strip,
// e leva a query junto (uma URL de mesa guardada nos favoritos tem parâmetros).
func alvoOriginal(r *http.Request) string {
	if r.RequestURI != "" && strings.HasPrefix(r.RequestURI, "/") {
		return r.RequestURI
	}
	return "/"
}

// requireAdmin gates the administration routes, and runs AFTER requireAuth —
// it reads the identity that middleware attached. The answer comes from
// ADMIN_EMAILS in the environment file, so the only way to gain it is editing
// that file on the host: no request can grant it (ALE-120).
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).IsAdmin {
			plataforma.WriteError(w, http.StatusForbidden, "Admin only")
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
