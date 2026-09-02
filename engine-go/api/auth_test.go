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

// `TestSignVerifyRoundtrip` saiu na ALE-187: assinar e verificar o próprio
// token é exercitado por TODA chamada `authed()` da suíte, centenas de vezes
// por corrida. O vizinho abaixo fica, e não é o mesmo: ele afirma a RECUSA
// de um token assinado com outro segredo, que nenhum caminho feliz cobre.
//
// O `TestParseExpiry` mais abaixo também FICA, contra o que a issue pedia:
// ele não é encanamento. O `parseExpiry` é parser escrito à mão, com dois
// fallbacks para `sessionTTL` — vazio e número inválido —, e um
// `JWT_EXPIRES_IN` malformado caindo em silêncio para o padrão é o tipo de
// coisa que ninguém descobre olhando.

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
