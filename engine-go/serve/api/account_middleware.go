package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"t20engine/infra/platform"
)

type ctxKey int

const userCtxKey ctxKey = iota

// requireAuth confere a sessão (cookie ou Bearer), carrega o usuário e guarda o
// `AuthUser` no contexto. Token que verifica contra usuário apagado é recusado.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.accountRules().sessionUser(r)
		if err != nil {
			platform.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// `errNoSession` e `errUserGone` são separados porque o SEGUNDO vale ser dito em
// voz alta: token que verifica contra usuário que não existe mais é cookie
// velho, não cookie forjado, e a mensagem é o único lugar onde essa diferença
// aparece.
var (
	errNoSession = errors.New("Unauthorized")
	errUserGone  = errors.New("User no longer exists")
)

// sessionUser resolve a identidade que a requisição carrega, SEM o middleware.
//
// A porta precisa perguntar "tem alguém logado?" para escolher entre desenhar a
// tela de entrar e redirecionar — pergunta que o middleware só saberia responder
// recusando a requisição.
func (a accountRules) sessionUser(r *http.Request) (AuthUser, error) {
	token := a.extractToken(r)
	if token == "" {
		return AuthUser{}, errNoSession
	}
	sub, err := a.verifyToken(token)
	if err != nil {
		return AuthUser{}, errNoSession
	}
	user, err := a.queries.GetUserByID(r.Context(), sub)
	if err != nil {
		return AuthUser{}, errUserGone
	}
	return a.authUser(user), nil
}

// requirePage é o `requireAuth` das PÁGINAS: quem não tem sessão vai para a
// porta, lembrando para onde ia.
//
// A diferença não é cosmética. O `requireAuth` responde um JSON 401, que é a
// resposta certa para quem chama a API e a errada para quem digitou uma URL: o
// jogador veria `{"statusCode":401}` numa tela branca em vez da tela de entrar.
//
// Vale para TODA página do Datastar, e não só para o Hub.
func (s *Server) requirePage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.accountRules().sessionUser(r)
		if err != nil {
			http.Redirect(w, r, "/entrar?redirect="+url.QueryEscape(alvoOriginal(r)), http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}

// alvoOriginal é a URL que o navegador pediu, com a query.
//
// `RequestURI` e não `URL.Path` por causa da QUERY: `URL.Path` a descarta, e uma
// URL de mesa guardada nos favoritos tem parâmetros. Perdê-los devolveria a
// pessoa a uma tela certa com o estado errado — e ninguém desconfia de uma tela
// certa.
func alvoOriginal(r *http.Request) string {
	if r.RequestURI != "" && strings.HasPrefix(r.RequestURI, "/") {
		return r.RequestURI
	}
	return "/"
}

// requireAdmin tranca as rotas de administração, e roda DEPOIS do `requireAuth`
// — ele lê a identidade que aquele middleware pendurou. A resposta vem do
// `ADMIN_EMAILS` do arquivo de ambiente, então o único jeito de ganhar o papel é
// editar esse arquivo no hospedeiro: requisição nenhuma o concede.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).IsAdmin {
			platform.WriteError(w, http.StatusForbidden, "Admin only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a accountRules) extractToken(r *http.Request) string {
	if c, err := r.Cookie(a.cfg.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return extractBearer(r.Header.Get("Authorization"))
}

// currentUser devolve o `AuthUser` pendurado pelo `requireAuth`, ou o valor zero.
func currentUser(r *http.Request) AuthUser {
	u, _ := r.Context().Value(userCtxKey).(AuthUser)
	return u
}
