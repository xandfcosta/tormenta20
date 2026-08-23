package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"t20engine/db/sqlcgen"
)

// O que a POÇÃO faz, pelo router de verdade (ALE-186, bloco 1).
//
// `consume.go` não tinha teste nenhum, e é onde mora a decisão registrada em
// [[spell_engine_deferred]]: catalisador é DECREMENTO INSTANTÂNEO até o motor
// de magias chegar. O que se prova aqui é o que a mesa observa — a poção some
// do inventário, o PV sobe e para no máximo, e o inventário é de quem o abriu.
// A conta do dado (2d4 → média 5) pertence ao `rollAverage` e está provada lá.

func seedConsumable(t *testing.T, s *Server, charID int64, catalogID, name string, qty int64) int64 {
	t.Helper()
	it, err := s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: charID, Catalogid: sql.NullString{String: catalogID, Valid: true},
		Name: name, Quantity: qty, Slots: 0.5, Improvements: "[]", Createdat: nowISO(),
	})
	if err != nil {
		t.Fatalf("semear item %q: %v", name, err)
	}
	return it.ID
}

func consumePath(charID, itemID int64) string {
	return "/characters/" + id64(charID) + "/items/" + id64(itemID) + "/consume"
}

func decodeConsume(t *testing.T, rec *httptest.ResponseRecorder) consumeResult {
	t.Helper()
	var out consumeResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("resposta não é um consumeResult (%s): %v", rec.Body.String(), err)
	}
	return out
}

func itemQuantity(t *testing.T, s *Server, itemID int64) (int64, bool) {
	t.Helper()
	row, err := s.queries.GetItem(context.Background(), itemID)
	if err == sql.ErrNoRows {
		return 0, false
	}
	if err != nil {
		t.Fatalf("ler item %d: %v", itemID, err)
	}
	return row.Quantity, true
}

func TestConsumeDecrementsExactlyOneAndHeals(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	// 12/20 de PV com bálsamo (2d4, média 5) cabe SEM encostar no teto: é o
	// caso que separa "curou" de "curou até o máximo", que o teste ao lado pega.
	char := seedCharacter(t, s, owner, "Guerreiro", 12, 20, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 3)

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (corpo %q)", rec.Code, rec.Body.String())
	}
	got := decodeConsume(t, rec)
	if got.Item.Quantity != 2 || got.Item.Removed {
		t.Fatalf("resposta diz quantidade %d removido=%v, esperado 2 e falso", got.Item.Quantity, got.Item.Removed)
	}
	if got.HpCurrent != 17 {
		t.Fatalf("PV = %d, esperado 17 (12 + os 5 de média de 2d4)", got.HpCurrent)
	}
	// O que a resposta diz e o que o banco guarda têm de ser a mesma coisa: a
	// ficha recarrega do banco, e uma resposta otimista que mentisse só
	// apareceria no F5 da próxima sessão.
	if qty, alive := itemQuantity(t, s, item); !alive || qty != 2 {
		t.Fatalf("no banco: quantidade %d viva=%v, esperado 2 e vivo", qty, alive)
	}
}

func TestConsumeLastUnitRemovesTheItem(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 12, 20, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 1)

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (corpo %q)", rec.Code, rec.Body.String())
	}
	if got := decodeConsume(t, rec); !got.Item.Removed {
		t.Fatalf("a última dose devia sair da mochila, veio %+v", got.Item)
	}
	if _, alive := itemQuantity(t, s, item); alive {
		t.Fatal("a última dose continua no banco — a mochila mostraria uma poção que não existe")
	}
}

func TestConsumeClampsGainAtMaximum(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 18, 20, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 1)

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if got := decodeConsume(t, rec); got.HpCurrent != 20 {
		t.Fatalf("PV = %d, esperado 20: 18 + 5 não pode passar do máximo", got.HpCurrent)
	}
}

func TestConsumeUsesRolledValueWhenTheTableRolls(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 10, 30, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 1)

	// A média é o padrão de quem não quer rolar; quem rola manda o resultado.
	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), `{"hpRolled":8}`)

	if got := decodeConsume(t, rec); got.HpCurrent != 18 {
		t.Fatalf("PV = %d, esperado 18 (10 + o 8 rolado na mesa)", got.HpCurrent)
	}
}

func TestConsumeCreatesTheSceneEffect(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Bardo", 10, 10, 0, 0)
	item := seedConsumable(t, s, char, "cosmetico", "Cosmético", 1)

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	got := decodeConsume(t, rec)
	if got.Effect == nil {
		t.Fatal("o cosmético dura a cena inteira e não devolveu efeito nenhum")
	}
	if got.Effect.Scope != "scene" || got.Effect.CatalogID != "cosmetico" {
		t.Fatalf("efeito = %+v, esperado escopo de cena para o cosmético", got.Effect)
	}
}

func TestConsumeRejectsAStranger(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	stranger := seedUser(t, s, "estranho@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 12, 20, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 2)

	rec := authed(t, s, stranger, http.MethodPost, consumePath(char, item), "{}")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, esperado 403: a mochila é de quem a carrega", rec.Code)
	}
	if qty, _ := itemQuantity(t, s, item); qty != 2 {
		t.Fatalf("quantidade = %d, esperado 2: o 403 não pode ter bebido a poção", qty)
	}
}

func TestConsumeRefusesWhatIsNotConsumable(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 12, 20, 0, 0)
	item := seedConsumable(t, s, char, "espada-longa", "Espada longa", 1)

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400 ao beber uma espada (corpo %q)", rec.Code, rec.Body.String())
	}
	if _, alive := itemQuantity(t, s, item); !alive {
		t.Fatal("a espada sumiu da mochila ao ser recusada")
	}
}

func TestConsumeRefusesTheSecondPortionOfTheDay(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 10, 40, 0, 0)
	item := seedConsumable(t, s, char, "macarrao-de-yuvalin", "Macarrão de Yuvalin", 2)

	first := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")
	if first.Code != http.StatusOK {
		t.Fatalf("a primeira porção falhou: %d (%s)", first.Code, first.Body.String())
	}

	second := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if second.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400: o catálogo marca este prato como uma vez por dia", second.Code)
	}
	if qty, _ := itemQuantity(t, s, item); qty != 1 {
		t.Fatalf("quantidade = %d, esperado 1: a porção recusada não pode ter sido comida", qty)
	}
}

func TestConsumeAllowsThePortionAgainAfterTheDayEnds(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 10, 40, 0, 0)
	item := seedConsumable(t, s, char, "macarrao-de-yuvalin", "Macarrão de Yuvalin", 2)

	authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")
	// A outra metade da regra: o marcador é de ESCOPO DE DIA, e encerrar o dia
	// o limpa. Sem isto, o conserto acima teria trocado "come o dia inteiro"
	// por "nunca mais come" — que é pior, porque some sem aviso.
	//
	// Chama o HELPER de domínio e não a rota HTTP: o que este teste protege é o
	// marcador do consumível, e a rota carrega uma autorização que não é assunto
	// dele — desde a ALE-223 ela pede um MESTRE em sessão viva, e montar uma
	// mesa aqui só para encerrar um dia mediria a regra errada.
	if status, err := s.endDay(context.Background(), AuthUser{ID: owner}, char); err != nil {
		t.Fatalf("encerrar o dia falhou: %d (%v)", status, err)
	}

	rec := authed(t, s, owner, http.MethodPost, consumePath(char, item), "{}")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200: amanhã se come de novo (corpo %q)", rec.Code, rec.Body.String())
	}
}
