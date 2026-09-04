package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"t20engine/plataforma"
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
		Name: name, Quantity: qty, Slots: 0.5, Improvements: "[]", Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear item %q: %v", name, err)
	}
	return it.ID
}

// consumeItem chama a REGRA direto, e não a rota.
//
// Ela batia em `POST /characters/{id}/items/{itemId}/consume`, que saiu na
// ALE-277 com as outras sessenta e nove rotas sem consumidor. O que estes casos
// prendem nunca foi o transporte: é a baixa de UMA dose, a cura presa no
// máximo, o efeito de cena e a porção diária. **Teste de regra vive junto da
// regra**, e o caminho é o mesmo que a Mochila da ficha usa pelo `ConsumeItem`
// da porta.
func consumeItem(t *testing.T, s *Server, charID, itemID int64, pv, pm *int64) (doseUsed, error) {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), charID)
	if err != nil {
		t.Fatalf("ler personagem %d: %v", charID, err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	return s.sheetRules().consumeItemForCharacter(req, row, itemID, pv, pm)
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

	dose, errDose := consumeItem(t, s, char, item, nil, nil)

	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}
	got := dose.consumeResult
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

	dose, errDose := consumeItem(t, s, char, item, nil, nil)

	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}
	if got := dose.consumeResult; !got.Item.Removed {
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

	dose, errDose := consumeItem(t, s, char, item, nil, nil)
	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}

	if got := dose.consumeResult; got.HpCurrent != 20 {
		t.Fatalf("PV = %d, esperado 20: 18 + 5 não pode passar do máximo", got.HpCurrent)
	}
}

func TestConsumeUsesRolledValueWhenTheTableRolls(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 10, 30, 0, 0)
	item := seedConsumable(t, s, char, "balsamo-restaurador", "Bálsamo restaurador", 1)

	// A média é o padrão de quem não quer rolar; quem rola manda o resultado.
	pvRolado := int64(8)
	dose, errDose := consumeItem(t, s, char, item, &pvRolado, nil)
	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}

	if got := dose.consumeResult; got.HpCurrent != 18 {
		t.Fatalf("PV = %d, esperado 18 (10 + o 8 rolado na mesa)", got.HpCurrent)
	}
}

func TestConsumeCreatesTheSceneEffect(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Bardo", 10, 10, 0, 0)
	item := seedConsumable(t, s, char, "cosmetico", "Cosmético", 1)

	dose, errDose := consumeItem(t, s, char, item, nil, nil)
	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}

	got := dose.consumeResult
	if got.Effect == nil {
		t.Fatal("o cosmético dura a cena inteira e não devolveu efeito nenhum")
	}
	if got.Effect.Scope != "scene" || got.Effect.CatalogID != "cosmetico" {
		t.Fatalf("efeito = %+v, esperado escopo de cena para o cosmético", got.Effect)
	}
}

// Aqui morava o TestConsumeRejectsAStranger, que provava o 403 de quem não é
// dono da mochila. Ele morreu com a rota na ALE-277, e a garantia não: a POSSE é
// do TRANSPORTE, e a cena da ficha a prende no próprio comando — o
// `characterFor` é o gargalo único por onde toda rota de personagem passa. Uma
// regra, uma camada.

func TestConsumeRefusesWhatIsNotConsumable(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	char := seedCharacter(t, s, owner, "Guerreiro", 12, 20, 0, 0)
	item := seedConsumable(t, s, char, "espada-longa", "Espada longa", 1)

	dose, errDose := consumeItem(t, s, char, item, nil, nil)

	if errDose == nil {
		t.Fatalf("status = — e não foi recusado (dose %+v)", dose)
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

	if _, first := consumeItem(t, s, char, item, nil, nil); first != nil {
		t.Fatalf("a primeira porção falhou: %v", first)
	}

	_, second := consumeItem(t, s, char, item, nil, nil)

	if second == nil {
		t.Fatal("a segunda porção passou: o catálogo marca este prato como uma vez por dia")
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

	consumeItem(t, s, char, item, nil, nil)
	// A outra metade da regra: o marcador é de ESCOPO DE DIA, e encerrar o dia
	// o limpa. Sem isto, o conserto acima teria trocado "come o dia inteiro"
	// por "nunca mais come" — que é pior, porque some sem aviso.
	//
	// Chama o HELPER de domínio e não a rota HTTP: o que este teste protege é o
	// marcador do consumível, e a rota carrega uma autorização que não é assunto
	// dele — desde a ALE-223 ela pede um MESTRE em sessão viva, e montar uma
	// mesa aqui só para encerrar um dia mediria a regra errada.
	if status, err := s.tableRules().endDay(context.Background(), AuthUser{ID: owner}, char); err != nil {
		t.Fatalf("encerrar o dia falhou: %d (%v)", status, err)
	}

	_, errDose := consumeItem(t, s, char, item, nil, nil)

	if errDose != nil {
		t.Fatalf("a dose foi recusada: %v", errDose)
	}
}
