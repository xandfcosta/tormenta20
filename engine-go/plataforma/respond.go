package plataforma

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// NullToPtr converts a nullable TEXT column into a *string (nil → JSON null).
func NullToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullString e NullBool são o caminho de VOLTA do `NullToPtr`: um ponteiro que
// pode ser nulo virando a coluna anulável que o sqlc espera.
//
// Elas moram aqui desde a ALE-278 porque a ficha em Datastar precisou das duas e
// não pode alcançar o `api`. O par com o `NullToPtr` é o argumento: adaptar
// `database/sql` na borda é infraestrutura sem uma palavra de domínio, e ter a
// ida num pacote e a volta noutro é a forma que faz as duas divergirem.
//
// NullBool grava BOOLEANO em coluna INTEIRA, que é como o SQLite guarda um
// `trained`.
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func NullBool(p *bool) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	var n int64
	if *p {
		n = 1
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

// WriteJSON serializes body as JSON with the given status. A nil body writes just
// the status line (204 responses). HTML escaping is off so `<`, `>`, `&` inside
// the stored JSON-string columns (modifiers, choices) reach the client verbatim
// instead of as \u003c escapes.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(body)
	}
}

// WriteError emits the plain error envelope: {"statusCode","message"}. When a
// failure also carries per-field detail, use WriteFieldError (validate.go) —
// this one deliberately stays the minimal shape.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{"statusCode": status, "message": message})
}
