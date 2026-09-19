package api

import (
	"net/http"
	"strings"

	"t20engine/infra/config"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/httpio"
)

// AuthUser é o contrato de identidade devolvido ao cliente. `Name` é nulo quando
// não foi preenchido.
type AuthUser struct {
	ID    int64   `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
	// IsAdmin é DERIVADO do ADMIN_EMAILS a cada pedido, nunca guardado: o papel
	// não tem linha para envelhecer, e é isto que a tela lê para mostrar a porta
	// da administração.
	IsAdmin bool `json:"isAdmin"`
}

func (a accountRules) authUser(u sqlcgen.User) AuthUser {
	out := AuthUser{ID: u.ID, Email: u.Email, IsAdmin: a.cfg.IsAdmin(u.Email)}
	if u.Name.Valid {
		out.Name = &u.Name.String
	}
	return out
}

// issueSession põe o biscoito da sessão que o caso de uso assinou. Devolve falso
// (depois de escrever um 500) quando a assinatura falha.
//
// A repartição é o ponto: QUANTO a sessão dura é a mesma decisão que assina o
// token, e por isso o número vem do caso de uso; o que é `HttpOnly`, `Secure` e
// `SameSite` é de quem responde HTTP, e por isso mora aqui.
func (a accountRules) issueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	token, err := a.gate.SignSession(user)
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not sign session")
		return false
	}
	http.SetCookie(w, sessionCookie(a.cfg, token, int(a.gate.SessionTTL().Seconds())))
	return true
}

// Ela recebe a CONFIGURAÇÃO em vez de pendurar no `*Server`: uma função que só
// precisa de dois campos não tem razão para exigir um servidor inteiro.
func sessionCookie(cfg config.Config, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     cfg.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func extractBearer(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
