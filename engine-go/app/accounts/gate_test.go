package accounts

import (
	"testing"
	"time"

	"t20engine/infra/config"
	"t20engine/infra/db/sqlcgen"
)

// DUAS GARANTIAS DA SESSÃO, e as duas vieram do `serve/api` junto com a regra
// (ALE-349). Nenhuma das duas precisa de banco: o portão só usa a configuração
// para assinar e conferir.

// portao monta um portão SEM banco. Assinar e conferir não tocam em `queries`
// nem em `db`, e é por isso que este caso não paga uma migração.
func portao(segredo string) Gate {
	return NewGate(nil, nil, config.Config{
		JWTSecret: segredo, JWTExpiresIn: "7d", CookieName: "t20_session",
	})
}

// A ida e volta — assinar e conferir o próprio token — é exercitada por TODA
// chamada autenticada da suíte, centenas de vezes por corrida, e por isso não
// tem caso próprio. O que nenhum caminho feliz cobre é a RECUSA de um token
// assinado com outro segredo.
func TestASessionSignedWithAnotherSecretIsRefused(t *testing.T) {
	tok, err := portao("real").SignSession(sqlcgen.User{ID: 1, Email: "x@y.com"})
	if err != nil {
		t.Fatalf("assinar a sessão: %v", err)
	}
	if _, err := portao("forjado").UserOfSession(tok); err == nil {
		t.Fatal("uma sessão assinada com outro segredo foi aceita")
	}
}

// O `parseExpiry` é parser escrito à mão, com dois recuos para o padrão — vazio
// e número inválido. Um `JWT_EXPIRES_IN` malformado caindo em silêncio para sete
// dias é o tipo de coisa que ninguém descobre olhando.
func TestParseExpiry(t *testing.T) {
	casos := map[string]time.Duration{
		"7d":  7 * 24 * time.Hour,
		"12h": 12 * time.Hour,
		"30m": 30 * time.Minute,
		"":    sessionTTL,
		"abc": sessionTTL,
	}
	for entrada, esperado := range casos {
		if veio := parseExpiry(entrada); veio != esperado {
			t.Errorf("parseExpiry(%q) = %v, esperado %v", entrada, veio, esperado)
		}
	}
}
